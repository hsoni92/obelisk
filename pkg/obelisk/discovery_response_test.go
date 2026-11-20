package obelisk

import (
	"strings"
	"testing"
	"time"
)

func TestBuildProjectDiscovery_FiltersMembership(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "db.example.com")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{
		{ID: "a", Type: "postgres", Enabled: true, Credentials: InstanceCredentials{Username: "u", Password: "p", Database: "d"}},
		{ID: "b", Type: "redis", Enabled: true, Credentials: InstanceCredentials{Password: "r"}},
	}
	spec.Projects = []Project{{
		ID:          "prj1",
		Name:        "app",
		InstanceIDs: []string{"postgres/a"},
	}}
	out := BuildProjectDiscovery(spec, spec.Projects[0], nil, time.Now(), ProjectDiscoveryExtras{}, "")
	if len(out.Instances) != 1 || out.Instances[0].Type != "postgres" {
		t.Fatalf("got %+v", out.Instances)
	}
}

func TestBuildProjectDiscovery_ExposureMember(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "edge.example.com")
	spec := DefaultSpec("vps")
	spec.OpenRouterGateway.Enabled = true
	spec.Projects = []Project{{
		ID:          "app",
		Name:        "app",
		InstanceIDs: []string{ExposureOpenRouterMember},
	}}
	out := BuildProjectDiscovery(spec, spec.Projects[0], nil, time.Now(), ProjectDiscoveryExtras{
		OpenRouterGatewayToken: "gw_secret",
	}, "")
	if len(out.Instances) != 1 || out.Instances[0].Type != "openrouter" {
		t.Fatalf("got %+v", out.Instances)
	}
	if !strings.Contains(out.Instances[0].Use, "gw_secret") {
		t.Fatalf("use=%q", out.Instances[0].Use)
	}
}

func TestValidateProjectInstanceRefs(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{ID: "x", Type: "redis", Enabled: true}}
	p := Project{Name: "p", ID: "1", InstanceIDs: []string{"redis/x"}}
	if err := ValidateProjectInstanceRefs(spec, p); err != nil {
		t.Fatal(err)
	}
	p.InstanceIDs = []string{"postgres/missing"}
	if err := ValidateProjectInstanceRefs(spec, p); err == nil {
		t.Fatal("expected error")
	}
	p.InstanceIDs = []string{ExposureOpenRouterMember}
	if err := ValidateProjectInstanceRefs(spec, p); err != nil {
		t.Fatal(err)
	}
}