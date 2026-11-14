package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// ObserveServiceStates reports cluster state without running helm upgrade.
func (r *Reconciler) ObserveServiceStates(ctx context.Context, spec obelisk.Spec) map[string]string {
	out := map[string]string{}
	for _, inst := range spec.Instances {
		key := inst.StateKey()
		if !inst.Enabled {
			out[key] = "stopped"
			continue
		}
		out[key] = "pending"
		if r.simulateMode() {
			out[key] = "running"
			continue
		}
		st, err := r.helmReleaseStatus(ctx, inst.ReleaseName())
		if err != nil {
			continue
		}
		switch st {
		case "deployed":
			out[key] = "running"
		case "pending-install", "pending-upgrade", "pending-rollback":
			out[key] = "pending"
		case "failed", "uninstalling":
			out[key] = "degraded"
		default:
			if st != "" {
				out[key] = st
			}
		}
	}
	return out
}

func (r *Reconciler) helmReleaseStatus(ctx context.Context, release string) (string, error) {
	ns := r.Namespace
	if ns == "" {
		ns = "obelisk"
	}
	cmd := exec.CommandContext(ctx, "helm", "status", release, "-n", ns, "-o", "json")
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
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

// ReconcileAsync runs reconcile in the background (single global queue).
func ReconcileAsync(rec Reconciler) {
	go func() {
		_, _ = rec.Reconcile(context.Background())
	}()
}

func (r *Reconciler) TriggerMessage() string {
	return fmt.Sprintf("reconcile queued for %s", r.ConfigPath)
}