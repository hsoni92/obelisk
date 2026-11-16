package obelisk

import "testing"

func TestFindInstanceByTypeAndID(t *testing.T) {
	spec := Spec{Instances: []ServiceInstance{
		{ID: "default", Type: "postgres", Enabled: false},
		{ID: "default", Type: "redis", Enabled: true},
	}}
	inst, _, ok := FindInstanceByTypeAndID(&spec, "redis", "default")
	if !ok || inst.Type != "redis" {
		t.Fatalf("got %#v ok=%v", inst, ok)
	}
	_, idx, ok := FindInstance(&spec, "default")
	if !ok || spec.Instances[idx].Type != "postgres" {
		t.Fatalf("FindInstance still returns first match")
	}
}