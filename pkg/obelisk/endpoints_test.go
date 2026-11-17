package obelisk

import (
	"strings"
	"testing"
)

func TestBuildConnections_Instance(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "db.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "default", Type: "postgres", Enabled: true,
		Credentials: InstanceCredentials{Username: "u", Password: "p", Database: "d"},
	}}
	states := map[string]string{"postgres/default": "running"}
	c := BuildConnections(spec, states)
	if len(c.Instances) != 1 || c.Instances[0].Postgres == "" {
		t.Fatalf("got %+v", c)
	}
	if c.Instances[0].Use != c.Instances[0].Postgres {
		t.Fatalf("use = Postgres URL, got %q", c.Instances[0].Use)
	}
}

func TestBuildConnections_EnabledNotRunningStillShares(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "vps.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "pqr", Type: "rustfs", Enabled: true,
		Credentials: InstanceCredentials{AccessKey: "obelisk", SecretKey: "s3cr3t"},
	}}
	c := BuildConnections(spec, map[string]string{"rustfs/pqr": "stopped"})
	if len(c.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %+v", c)
	}
	use := c.Instances[0].Use
	if !strings.Contains(use, "S3_ENDPOINT=http://s3-pqr.vps.example.com") ||
		!strings.Contains(use, "AWS_ACCESS_KEY_ID=") ||
		!strings.Contains(use, "AWS_SECRET_ACCESS_KEY=") || !strings.Contains(use, "obelisk") || !strings.Contains(use, "s3cr3t") {
		t.Fatalf("unexpected use block:\n%s", use)
	}
}

func TestInstanceUse_QdrantIncludesKey(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "iq1", Type: "qdrant", Enabled: true,
		Credentials: InstanceCredentials{AccessKey: "qkey"},
	}}
	c := BuildConnections(spec, nil)
	use := c.Instances[0].Use
	if !strings.Contains(use, "QDRANT_API_KEY=qkey") {
		t.Fatalf("unexpected use:\n%s", use)
	}
}

func TestBuildConnections_DisabledOmitted(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "x", Type: "redis", Enabled: false,
		Credentials: InstanceCredentials{Password: "p"},
	}}
	c := BuildConnections(spec, nil)
	if len(c.Instances) != 0 {
		t.Fatalf("disabled instances should not appear: %+v", c)
	}
}