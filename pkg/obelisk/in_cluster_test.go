package obelisk

import (
	"strings"
	"testing"
	"time"
)

func TestBuildConnections_InClusterPostgres(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "vps.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "calm-otter", Type: "postgres", Enabled: true,
		Credentials: InstanceCredentials{Username: "u", Password: "p", Database: "app"},
	}}
	c := BuildConnections(spec, nil)
	if len(c.Instances) != 1 {
		t.Fatalf("instances: %+v", c.Instances)
	}
	ic := c.Instances[0]
	if ic.InCluster == nil {
		t.Fatal("missing in_cluster block")
	}
	wantHost := "obelisk-postgres-calm-otter.obelisk.svc.cluster.local"
	if ic.InCluster.Host != wantHost {
		t.Fatalf("host=%q want %q", ic.InCluster.Host, wantHost)
	}
	if !strings.Contains(ic.InCluster.Postgres, wantHost+":5432") || !strings.Contains(ic.InCluster.Postgres, "sslmode=disable") {
		t.Fatalf("postgres=%q", ic.InCluster.Postgres)
	}
	if strings.Contains(c.Instances[0].Postgres, "9443") && !strings.Contains(ic.InCluster.Postgres, "9443") {
		// edge vs in-cluster ports
	} else if ic.InCluster.Postgres == ic.Postgres {
		t.Fatal("in-cluster postgres should differ from edge on vps")
	}
}

func TestInstanceConnectionForAudience_InCluster(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "vps.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "r1", Type: "redis", Enabled: true,
		Credentials: InstanceCredentials{Password: "secret"},
	}}
	ic := BuildConnections(spec, nil).Instances[0]
	promoted := InstanceConnectionForAudience(ic, ConnectionAudienceInCluster)
	if promoted.Redis != ic.InCluster.Redis {
		t.Fatalf("promoted redis=%q in_cluster=%q", promoted.Redis, ic.InCluster.Redis)
	}
	// Redis chart is TLS-only when the VPS edge is on, so the URL must be rediss://.
	if !strings.HasPrefix(promoted.Redis, "rediss://") {
		t.Fatalf("in-cluster redis must be rediss:// on a TLS-only chart: %q", promoted.Redis)
	}
	if promoted.TLS == nil || !promoted.TLS.Enabled || promoted.TLS.Verify {
		t.Fatalf("tls descriptor=%+v", promoted.TLS)
	}
}

func TestInCluster_MongoCarriesTLSOptions(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "vps.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "m1", Type: "mongodb", Enabled: true,
		Credentials: InstanceCredentials{Username: "u", Password: "p", Database: "d"},
	}}
	ic := BuildConnections(spec, nil).Instances[0]
	for _, u := range []string{ic.InCluster.MongoDB, ic.InCluster.Databases[0].URL, ic.MongoDB} {
		if !strings.Contains(u, "tls=true") || !strings.Contains(u, "tlsAllowInvalidCertificates=true") {
			t.Fatalf("mongo url missing tls options: %q", u)
		}
	}
}

func TestBuildProjectDiscovery_AudienceInCluster(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "vps.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "a", Type: "postgres", Enabled: true,
		Credentials: InstanceCredentials{Username: "u", Password: "p", Database: "d"},
	}}
	spec.Projects = []Project{{
		ID: "p1", Name: "co-located", InstanceIDs: []string{"postgres/a"},
	}}
	out := BuildProjectDiscovery(spec, spec.Projects[0], nil, time.Now(), ProjectDiscoveryExtras{}, ConnectionAudienceInCluster)
	if !strings.Contains(out.Notes, "In-cluster:") {
		t.Fatalf("notes=%q", out.Notes)
	}
	if !strings.Contains(out.Instances[0].Postgres, "svc.cluster.local") {
		t.Fatalf("postgres=%q", out.Instances[0].Postgres)
	}
}
