package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/exposure"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

var reconcileMu sync.Mutex

// Reconciler applies desired state to the cluster (Helm releases).
type Reconciler struct {
	ConfigPath string
	Namespace  string
	HelmRoot   string
}

type Report struct {
	ServiceStates map[string]string
	Messages      []string
}

func (r *Reconciler) Reconcile(ctx context.Context) (Report, error) {
	reconcileMu.Lock()
	defer reconcileMu.Unlock()

	spec, err := config.Load(r.ConfigPath)
	if err != nil {
		return Report{}, err
	}
	report := Report{ServiceStates: map[string]string{}}
	if r.reconcileDisabled() {
		report.Messages = append(report.Messages,
			"reconcile off (OBELISK_RECONCILE=off or helm/kubectl missing): desired state only")
		for _, inst := range spec.Instances {
			key := inst.StateKey()
			if !inst.Enabled {
				report.ServiceStates[key] = "stopped"
				continue
			}
			report.ServiceStates[key] = "pending"
		}
		fed, _ := r.federatedTargets(spec)
		if err := exposure.Reconcile(ctx, spec, r.HelmRoot, r.Namespace, fed, r.ConfigPath); err != nil {
			report.Messages = append(report.Messages, "exposure: "+err.Error())
		}
		return report, nil
	}
	if pruned := r.pruneOrphanReleases(ctx, spec); len(pruned) > 0 {
		report.Messages = append(report.Messages, "pruned orphan releases: "+strings.Join(pruned, ", "))
	}

	var reconcileErr error
	vpsEdge := obelisk.VPSEdge(spec)
	if !r.reconcileDisabled() && vpsEdge && exposure.NeedsNativeEdgeTLS(spec) {
		if err := exposure.ReconcileEdgeTLSSecret(ctx, r.Namespace); err != nil {
			report.Messages = append(report.Messages, "edge tls: "+err.Error())
			reconcileErr = err
		}
	}
	ordered := instancesReconcileOrder(spec.Instances)
	var disabled, enabled []obelisk.ServiceInstance
	for _, inst := range ordered {
		if inst.Enabled {
			enabled = append(enabled, inst)
		} else {
			disabled = append(disabled, inst)
		}
	}

	type instResult struct {
		key   string
		state string
		msg   string
		err   error
	}
	apply := func(res instResult) {
		report.ServiceStates[res.key] = res.state
		if res.msg != "" {
			report.Messages = append(report.Messages, res.msg)
		}
		if res.err != nil {
			if reconcileErr == nil {
				reconcileErr = res.err
			}
			report.Messages = append(report.Messages, fmt.Sprintf("%s: %v", res.key, res.err))
			report.ServiceStates[res.key] = "degraded"
		}
	}

	if len(disabled) > 0 {
		ch := make(chan instResult, len(disabled))
		var wg sync.WaitGroup
		for _, inst := range disabled {
			wg.Add(1)
			go func(inst obelisk.ServiceInstance) {
				defer wg.Done()
				state, msg, err := r.reconcileInstance(ctx, vpsEdge, inst)
				ch <- instResult{inst.StateKey(), state, msg, err}
			}(inst)
		}
		wg.Wait()
		close(ch)
		for res := range ch {
			apply(res)
		}
	}
	if len(enabled) > 0 {
		ch := make(chan instResult, len(enabled))
		var wg sync.WaitGroup
		for _, inst := range enabled {
			wg.Add(1)
			go func(inst obelisk.ServiceInstance) {
				defer wg.Done()
				state, msg, err := r.reconcileInstance(ctx, vpsEdge, inst)
				ch <- instResult{inst.StateKey(), state, msg, err}
			}(inst)
		}
		wg.Wait()
		close(ch)
		for res := range ch {
			apply(res)
		}
	}
	if spec.Deployment.Tailscale.Enabled {
		if err := r.ensureTailscale(ctx); err != nil {
			report.Messages = append(report.Messages, "tailscale: "+err.Error())
		}
	}
	fed, _ := r.federatedTargets(spec)
	if err := exposure.Reconcile(ctx, spec, r.HelmRoot, r.Namespace, fed, r.ConfigPath); err != nil {
		report.Messages = append(report.Messages, "exposure: "+err.Error())
		if reconcileErr == nil {
			reconcileErr = err
		}
	}
	return report, reconcileErr
}

// instancesReconcileOrder applies scale-down (disabled) instances before scale-up so
// a failing start does not block stopping other workloads — common after blueprint import.
func instancesReconcileOrder(insts []obelisk.ServiceInstance) []obelisk.ServiceInstance {
	out := append([]obelisk.ServiceInstance(nil), insts...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Enabled == out[j].Enabled {
			return false
		}
		return !out[i].Enabled
	})
	return out
}

func (r *Reconciler) reconcileInstance(ctx context.Context, vpsEdge bool, inst obelisk.ServiceInstance) (state, msg string, err error) {
	if !inst.Enabled {
		chart := fmt.Sprintf("%s/deploy/helm/services/%s", r.HelmRoot, inst.Type)
		release := inst.ReleaseName()
		ns := r.Namespace
		if ns == "" {
			ns = "obelisk"
		}
		valuesPath, cleanup, err := writeInstanceValues(vpsEdge, r.edgeTLSSecretReady(ctx), inst)
		if err != nil {
			return "degraded", "", err
		}
		defer cleanup()
		opts := helmUpgradeOpts{waitTimeout: 2 * time.Minute}
		if err := r.helmUpgradeInstall(ctx, inst.Type, chart, release, ns, valuesPath, opts); err != nil {
			return "degraded", "", err
		}
		st := r.classifyInstance(ctx, inst).State
		return st, fmt.Sprintf("%s scaled to 0 via helm", inst.StateKey()), nil
	}
	if err := r.helmUpgradeInstance(ctx, vpsEdge, inst); err != nil {
		return "degraded", "", err
	}
	st := r.classifyInstance(ctx, inst).State
	msg = fmt.Sprintf("%s release applied", inst.StateKey())
	if st == "running" {
		if err := r.ensureInstanceDatabases(ctx, inst); err != nil {
			if execAccessDenied(err) {
				return st, msg, nil
			}
			return "degraded", msg, fmt.Errorf("databases: %w", err)
		}
	}
	return st, msg, nil
}

func (r *Reconciler) helmUpgradeInstance(ctx context.Context, vpsEdge bool, inst obelisk.ServiceInstance) error {
	chart := fmt.Sprintf("%s/deploy/helm/services/%s", r.HelmRoot, inst.Type)
	release := inst.ReleaseName()
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	valuesPath, cleanup, err := writeInstanceValues(vpsEdge, r.edgeTLSSecretReady(ctx), inst)
	if err != nil {
		return err
	}
	defer cleanup()
	slog.Info("helm upgrade", "release", release, "chart", chart)
	return r.helmUpgradeInstall(ctx, inst.Type, chart, release, ns, valuesPath, helmUpgradeOpts{})
}

func (r *Reconciler) helmUpgrade(ctx context.Context, service string) error {
	// legacy path — unused; kept for callers migrating
	chart := fmt.Sprintf("%s/deploy/helm/services/%s", r.HelmRoot, service)
	release := "obelisk-" + service
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	slog.Info("helm upgrade", "release", release, "chart", chart)
	return r.helmUpgradeInstall(ctx, service, chart, release, ns, "", helmUpgradeOpts{})
}

func (r *Reconciler) helmScaleRelease(ctx context.Context, release string, replicas int) error {
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	cmd := exec.CommandContext(ctx, "kubectl", "scale", "statefulset", release,
		"-n", ns, fmt.Sprintf("--replicas=%d", replicas))
	cmd.Env = kubeconfigEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("scale statefulset failed", "release", release, "out", string(out), "err", err)
		return fmt.Errorf("scale %s: %w: %s", release, err, string(out))
	}
	return nil
}

func (r *Reconciler) helmScale(ctx context.Context, service string, replicas int) error {
	return r.helmScaleRelease(ctx, "obelisk-"+service, replicas)
}

func (r *Reconciler) edgeTLSSecretReady(ctx context.Context) bool {
	if r.reconcileDisabled() {
		return false
	}
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	cmd := exec.CommandContext(ctx, "kubectl", "get", "secret", "obelisk-edge-tls", "-n", ns)
	cmd.Env = kubeconfigEnv()
	return cmd.Run() == nil
}

func (r *Reconciler) ensureTailscale(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "tailscale", "status", "--json")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tailscale not connected: run tailscale up")
	}
	return nil
}

func SetServiceEnabled(spec *obelisk.Spec, name string, enabled bool) error {
	obelisk.NormalizeInstances(spec)
	found := false
	for i := range spec.Instances {
		if spec.Instances[i].Type != name {
			continue
		}
		if spec.Instances[i].ID == "default" {
			spec.Instances[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found && enabled {
		inst, err := obelisk.NewServiceInstance(name, obelisk.LegacyDefaultInstanceID, "default")
		if err != nil {
			return err
		}
		inst.ID = "default"
		inst.Credentials = obelisk.DefaultCredentialsForType(name)
		if name == "postgres" || name == "mysql" || name == "mongodb" {
			inst.Credentials.Password = "changeme-in-ui"
		}
		spec.Instances = append(spec.Instances, inst)
		found = true
	}
	if !found && !enabled {
		for i := range spec.Instances {
			if spec.Instances[i].Type == name {
				spec.Instances[i].Enabled = false
			}
		}
	}
	obelisk.SyncLegacyServiceFlags(spec)
	return nil
}
