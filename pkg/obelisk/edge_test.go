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

func TestBuildConnections_VPSPostgresEdge(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "sleek-eagle", Type: "postgres", Enabled: true,
		Credentials: InstanceCredentials{Password: "secret"},
	}}
	c := BuildConnections(spec, nil)
	want := "postgresql://obelisk:secret@postgres-sleek-eagle.178-105-35-230.sslip.io:9443/default?" + VPSPostgresURLQuery
	if c.Instances[0].Postgres != want {
		t.Fatalf("postgres %q", c.Instances[0].Postgres)
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
	want := "rediss://:secret@redis-pqr.178-105-35-230.sslip.io:9443/0"
	if c.Instances[0].Redis != want {
		t.Fatalf("redis %q", c.Instances[0].Redis)
	}
}

func TestNativeEdgeTLSPortNeverReserved(t *testing.T) {
	t.Setenv("OBELISK_TLS_PORT", "443")
	if got := NativeEdgeTLSPortNumber(); got != DefaultNativeEdgeTLSPort {
		t.Fatalf("reserved override accepted: %d", got)
	}
	t.Setenv("OBELISK_TLS_PORT", "9443")
	if got := NativeEdgeTLSPortNumber(); got != 9443 {
		t.Fatalf("override ignored: %d", got)
	}
}

func TestAdminHTTPPortNeverReserved(t *testing.T) {
	t.Setenv("OBELISK_HTTP_PORT", "80")
	if got := AdminHTTPPort(); got != DefaultAdminHTTPPort {
		t.Fatalf("reserved override accepted: %d", got)
	}
	t.Setenv("OBELISK_HTTP_PORT", "9080")
	if got := AdminHTTPPort(); got != 9080 {
		t.Fatalf("override ignored: %d", got)
	}
}

func TestReservedHostPortConflict(t *testing.T) {
	for _, port := range []int{80, 443} {
		if _, ok := ReservedHostPortConflict(port); !ok {
			t.Fatalf("port %d should be reserved", port)
		}
	}
	for _, port := range []int{9080, 9443} {
		if _, ok := ReservedHostPortConflict(port); ok {
			t.Fatalf("port %d should be allowed", port)
		}
	}
}