package controller

import (
	"testing"
)

func TestSimulateMode_ForcedCluster(t *testing.T) {
	t.Setenv("OBELISK_RECONCILE", "cluster")
	r := Reconciler{}
	if r.simulateMode() {
		t.Fatal("expected cluster mode")
	}
}

func TestSimulateMode_ForcedSimulate(t *testing.T) {
	t.Setenv("OBELISK_RECONCILE", "simulate")
	t.Setenv("OBELISK_DEV", "")
	r := Reconciler{}
	if !r.simulateMode() {
		t.Fatal("expected simulate mode")
	}
}