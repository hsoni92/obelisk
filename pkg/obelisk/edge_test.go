package obelisk

import "testing"

func TestPublicEdgeHost(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	inst := ServiceInstance{ID: "pqr", Type: "redis"}
	got := PublicEdgeHost(inst)
	want := "redis-pqr.178-105-35-230.sslip.io"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildConnections_VPSRedisEdge(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "pqr", Type: "redis", Enabled: true,
		Credentials: InstanceCredentials{Password: "secret"},
	}}
	c := BuildConnections(spec, nil)
	if len(c.Instances) != 1 {
		t.Fatalf("%+v", c)
	}
	want := "rediss://:secret@redis-pqr.178-105-35-230.sslip.io:443/0"
	if c.Instances[0].Redis != want {
		t.Fatalf("redis %q", c.Instances[0].Redis)
	}
}