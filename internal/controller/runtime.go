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

// simulateMode skips Helm/kubectl when local dev has no cluster, unless forced to cluster mode.
// OBELISK_RECONCILE=cluster forces real reconcile; simulate or OBELISK_DEV=1 forces skip.
// SimulateMode reports whether Helm/kubectl are skipped for this reconcile.
func (r *Reconciler) SimulateMode() bool {
	return r.simulateMode()
}

func (r *Reconciler) simulateMode() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OBELISK_RECONCILE"))) {
	case "cluster":
		return false
	case "simulate", "sim", "off", "dry-run":
		return true
	}
	if strings.TrimSpace(os.Getenv("OBELISK_DEV")) == "1" {
		return true
	}
	return !clusterToolsAvailable()
}