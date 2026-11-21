package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// RestartOutcome summarizes a best-effort workload restart.
type RestartOutcome struct {
	Method  string `json:"method"`
	Message string `json:"message"`
}

// RestartInstanceWorkload deletes or rolls pods for the instance release, then queues reconcile.
func (r *Reconciler) RestartInstanceWorkload(ctx context.Context, inst obelisk.ServiceInstance) RestartOutcome {
	if !inst.Enabled {
		return RestartOutcome{Method: "skipped", Message: "instance is stopped"}
	}
	stateKey := inst.StateKey()
	queueReconcile := func() {
		if e := GlobalEngine(); e != nil && e.Scheduler != nil {
			e.TriggerInstance(stateKey)
			return
		}
		ReconcileAsync(*r)
	}

	if r.reconcileDisabled() {
		queueReconcile()
		return RestartOutcome{
			Method:  "reconcile_only",
			Message: "cluster tools unavailable; reconcile queued",
		}
	}

	release := inst.ReleaseName()
	ns := r.namespace()

	cmd := exec.CommandContext(ctx, "kubectl", "rollout", "restart",
		"statefulset/"+release, "-n", ns)
	cmd.Env = kubeconfigEnv()
	if out, err := cmd.CombinedOutput(); err == nil {
		_ = out
		queueReconcile()
		return RestartOutcome{
			Method:  "statefulset_rollout",
			Message: fmt.Sprintf("rollout restart %s", release),
		}
	}

	cmd = exec.CommandContext(ctx, "kubectl", "delete", "pod", "-n", ns,
		"-l", "app="+release, "--wait=false", "--ignore-not-found=true")
	cmd.Env = kubeconfigEnv()
	if out, err := cmd.CombinedOutput(); err == nil && strings.TrimSpace(string(out)) != "" {
		queueReconcile()
		return RestartOutcome{
			Method:  "pod_delete",
			Message: fmt.Sprintf("deleted pods for %s", release),
		}
	}

	// Release missing or no pods yet — still reconcile to heal chart / scale.
	queueReconcile()
	slog.Info("instance restart: no workload; reconcile only", "release", release)
	return RestartOutcome{
		Method:  "reconcile_only",
		Message: "no running workload found; reconcile queued",
	}
}