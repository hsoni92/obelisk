package obelisk

import "testing"

func TestReleaseName_Default(t *testing.T) {
	i := ServiceInstance{ID: "default", Type: "postgres"}
	if i.ReleaseName() != "obelisk-postgres" {
		t.Fatal(i.ReleaseName())
	}
}

func TestNormalizeInstances_FromFlags(t *testing.T) {
	spec := DefaultSpec("local")
	spec.Services.Postgres.Enabled = true
	NormalizeInstances(&spec)
	if len(spec.Instances) != 1 || spec.Instances[0].Type != "postgres" {
		t.Fatalf("%+v", spec.Instances)
	}
}
