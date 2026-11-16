package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"sync"

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
		if err := exposure.ApplyFunnel(spec); err != nil {
			report.Messages = append(report.Messages, "exposure: "+err.Error())
		}
		return report, nil
	}
	var reconcileErr error
	for _, inst := range instancesReconcileOrder(spec.Instances) {
		state, msg, err := r.reconcileInstance(ctx, inst)
		report.ServiceStates[inst.StateKey()] = state
		if msg != "" {
			report.Messages = append(report.Messages, msg)
		}
		if err != nil {
			if reconcileErr == nil {
				reconcileErr = err
			}
			report.Messages = append(report.Messages,
				fmt.Sprintf("%s: %v", inst.StateKey(), err))
			report.ServiceStates[inst.StateKey()] = "degraded"
			continue
		}
	}
	if spec.Deployment.Profile == "local" && spec.Deployment.Tailscale.Enabled {
		if err := r.ensureTailscale(ctx); err != nil {
			report.Messages = append(report.Messages, "tailscale: "+err.Error())
		}
	}
	if err := exposure.ApplyFunnel(spec); err != nil {
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

func (r *Reconciler) reconcileInstance(ctx context.Context, inst obelisk.ServiceInstance) (state, msg string, err error) {
	if !inst.Enabled {
		chart := fmt.Sprintf("%s/deploy/helm/services/%s", r.HelmRoot, inst.Type)
		release := inst.ReleaseName()
		ns := r.Namespace
		if ns == "" {
			ns = "obelisk"
		}
		valuesPath, cleanup, err := writeInstanceValues(inst)
		if err != nil {
			return "degraded", "", err
		}
		defer cleanup()
		if err := r.helmUpgradeInstall(ctx, inst.Type, chart, release, ns, valuesPath); err != nil {
			return "degraded", "", err
		}
		st := r.observeInstanceState(ctx, inst)
		return st, fmt.Sprintf("%s scaled to 0 via helm", inst.StateKey()), nil
	}
	if err := r.helmUpgradeInstance(ctx, inst); err != nil {
		return "degraded", "", err
	}
	st := r.observeInstanceState(ctx, inst)
	return st, fmt.Sprintf("%s release applied", inst.StateKey()), nil
}

func (r *Reconciler) helmUpgradeInstance(ctx context.Context, inst obelisk.ServiceInstance) error {
	chart := fmt.Sprintf("%s/deploy/helm/services/%s", r.HelmRoot, inst.Type)
	release := inst.ReleaseName()
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	valuesPath, cleanup, err := writeInstanceValues(inst)
	if err != nil {
		return err
	}
	defer cleanup()
	slog.Info("helm upgrade", "release", release, "chart", chart)
	return r.helmUpgradeInstall(ctx, inst.Type, chart, release, ns, valuesPath)
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
	return r.helmUpgradeInstall(ctx, service, chart, release, ns, "")
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
		inst, err := obelisk.NewServiceInstance(name, "default")
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