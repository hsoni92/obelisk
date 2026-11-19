package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Instance states (UI + API):
//   - stopped:  desired off and no workload pods (quiescent)
//   - stopping: desired off but pods still terminating or helm still scaling down
//   - starting: desired on but not yet ready (helm pending or pods coming up)
//   - running:  desired on, helm deployed, pods ready
//   - degraded: helm failed or observation error with likely misalignment

// ObserveServiceStates reports cluster state without running helm upgrade.
func (r *Reconciler) ObserveServiceStates(ctx context.Context, spec obelisk.Spec) map[string]string {
	if e := GlobalEngine(); e != nil && e.Scheduler != nil {
		return e.Scheduler.ObserveStates(ctx, spec)
	}
	out := map[string]string{}
	for _, inst := range spec.Instances {
		out[inst.StateKey()] = r.classifyInstance(ctx, inst).State
	}
	return out
}

func (r *Reconciler) observeInstanceState(ctx context.Context, inst obelisk.ServiceInstance) string {
	if r.reconcileDisabled() {
		if inst.Enabled {
			return "starting"
		}
		return "stopped"
	}
	if !inst.Enabled {
		return r.observeDisabled(ctx, inst)
	}
	return r.observeEnabled(ctx, inst)
}

func (r *Reconciler) observeDisabled(ctx context.Context, inst obelisk.ServiceInstance) string {
	release := inst.ReleaseName()
	pods := r.PodsForRelease(ctx, release)
	if pods.Err != nil {
		return "degraded"
	}
	if pods.Total > 0 {
		return "stopping"
	}
	st, err := r.helmReleaseStatus(ctx, release)
	if err != nil || st == "" {
		return "stopped"
	}
	switch st {
	case "pending-install", "pending-upgrade", "pending-rollback", "uninstalling":
		return "stopping"
	case "failed":
		return "degraded"
	default:
		return "stopped"
	}
}

func (r *Reconciler) observeEnabled(ctx context.Context, inst obelisk.ServiceInstance) string {
	release := inst.ReleaseName()
	st, err := r.helmReleaseStatus(ctx, release)
	if err != nil {
		return "degraded"
	}
	if st == "" {
		return "starting"
	}
	switch st {
	case "failed", "uninstalling":
		return "degraded"
	case "pending-install", "pending-upgrade", "pending-rollback":
		return "starting"
	case "deployed":
		return r.stateWhenHelmDeployed(ctx, release)
	default:
		return "starting"
	}
}

// stateWhenHelmDeployed requires workload pods to exist and be ready (Helm deployed ≠ running).
func (r *Reconciler) stateWhenHelmDeployed(ctx context.Context, release string) string {
	pods := r.PodsForRelease(ctx, release)
	if pods.Err != nil {
		return "degraded"
	}
	if pods.Total == 0 {
		return "starting"
	}
	if pods.Ready < pods.Total {
		return "starting"
	}
	return "running"
}

func (r *Reconciler) helmReleaseStatus(ctx context.Context, release string) (string, error) {
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	cmd := exec.CommandContext(ctx, "helm", "status", release, "-n", ns, "-o", "json")
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.CombinedOutput()
	if err != nil {
		if HelmReleaseNotFound(raw, err) {
			return "", nil
		}
		return "", err
	}
	var doc struct {
		Info struct {
			Status string `json:"status"`
		} `json:"info"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	return strings.ToLower(doc.Info.Status), nil
}

// ReconcileAsync queues a full reconciliation pass (coalesced by the scheduler).
func ReconcileAsync(rec Reconciler) {
	if e := GlobalEngine(); e != nil {
		e.Trigger()
		return
	}
	go func() {
		report, err := rec.Reconcile(context.Background())
		if err != nil {
			slog.Warn("reconcile async", "err", err, "messages", report.Messages)
		}
	}()
}

func (r *Reconciler) TriggerMessage() string {
	return fmt.Sprintf("reconcile queued for %s", r.ConfigPath)
}