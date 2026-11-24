package controller

import (
	"os"
	"os/exec"
	"strings"
)

// clusterToolsAvailable reports whether helm and kubectl are on PATH.
func clusterToolsAvailable() bool {
	if _, err := exec.LookPath("helm"); err != nil {
		return false
	}
	if _, err := exec.LookPath("kubectl"); err != nil {
		return false
	}
	return true
}

func reconcileDisabledFromEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OBELISK_RECONCILE"))) {
	case "off", "dry-run", "observe", "observe-only", "simulate", "sim":
		return true
	case "cluster":
		return false
	}
	return !clusterToolsAvailable()
}

// ReconcileDisabled is true when the controller must not run Helm/kubectl (observe-only).
func ReconcileDisabled() bool {
	return reconcileDisabledFromEnv()
}

func (r *Reconciler) reconcileDisabled() bool {
	return reconcileDisabledFromEnv()
}

// SimulateMode reports whether Helm/kubectl are skipped (legacy name; prefer ReconcileDisabled).
func (r *Reconciler) SimulateMode() bool {
	return r.reconcileDisabled()
}
