package controller

import (
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestInstancesReconcileOrder_DisabledFirst(t *testing.T) {
	insts := []obelisk.ServiceInstance{
		{ID: "a", Type: "rustfs", Enabled: true},
		{ID: "b", Type: "qdrant", Enabled: false},
		{ID: "c", Type: "redis", Enabled: true},
		{ID: "d", Type: "postgres", Enabled: false},
	}
	got := instancesReconcileOrder(insts)
	if len(got) != len(insts) {
		t.Fatalf("len = %d", len(got))
	}
	var disabled, enabled int
	for i, inst := range got {
		if !inst.Enabled {
			disabled++
			if i >= 2 {
				t.Fatalf("disabled instance %s not in first half at i=%d", inst.ID, i)
			}
		} else {
			enabled++
			if i < 2 {
				t.Fatalf("enabled instance %s still in disabled block at i=%d", inst.ID, i)
			}
		}
	}
	if disabled != 2 || enabled != 2 {
		t.Fatalf("disabled=%d enabled=%d", disabled, enabled)
	}
}
