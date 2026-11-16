package controller

import (
	"testing"
)

func TestReconcileDisabled_ForcedCluster(t *testing.T) {
	t.Setenv("OBELISK_RECONCILE", "cluster")
	r := Reconciler{}
	if r.reconcileDisabled() {
		t.Fatal("expected cluster reconcile")
	}
}

func TestReconcileDisabled_Off(t *testing.T) {
	t.Setenv("OBELISK_RECONCILE", "off")
	t.Setenv("OBELISK_DEV", "")
	r := Reconciler{}
	if !r.reconcileDisabled() {
		t.Fatal("expected reconcile off")
	}
}

func TestReconcileDisabled_LegacySimulate(t *testing.T) {
	t.Setenv("OBELISK_RECONCILE", "simulate")
	r := Reconciler{}
	if !r.reconcileDisabled() {
		t.Fatal("expected simulate env to disable reconcile")
	}
}