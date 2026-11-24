package obelisk

import (
	"regexp"
	"testing"
)

var routeSlugShape = regexp.MustCompile(`^[a-z]+-[a-z]+$`)

func TestRandomRouteSlugShape(t *testing.T) {
	for i := 0; i < 32; i++ {
		s := randomRouteSlug()
		if !routeSlugShape.MatchString(s) {
			t.Fatalf("slug %q", s)
		}
		if !instanceIDValid(s) {
			t.Fatalf("invalid id %q", s)
		}
	}
}

func TestAllocateInstanceID_Unique(t *testing.T) {
	spec := DefaultSpec("vps")
	ids := map[string]struct{}{}
	for i := 0; i < 20; i++ {
		id := AllocateInstanceID(&spec)
		if !routeSlugShape.MatchString(id) {
			t.Fatalf("expected adjective-animal slug, got %q", id)
		}
		if _, ok := ids[id]; ok {
			t.Fatalf("duplicate %s", id)
		}
		ids[id] = struct{}{}
		spec.Instances = append(spec.Instances, ServiceInstance{ID: id, Type: "rustfs"})
	}
}

func TestRepairDuplicateInstanceIDs(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{
		{ID: LegacyDefaultInstanceID, Type: "postgres", DisplayName: "main"},
		{ID: LegacyDefaultInstanceID, Type: "redis", DisplayName: "cache"},
	}
	repairDuplicateInstanceIDs(&spec)
	if spec.Instances[0].ID != LegacyDefaultInstanceID {
		t.Fatalf("postgres keeps legacy id: %+v", spec.Instances[0])
	}
	if spec.Instances[1].ID != LegacyDefaultInstanceID {
		t.Fatalf("redis keeps legacy id across types: %+v", spec.Instances[1])
	}
	spec.Instances = append(spec.Instances, ServiceInstance{
		ID: LegacyDefaultInstanceID, Type: "redis", DisplayName: "replica",
	})
	repairDuplicateInstanceIDs(&spec)
	if spec.Instances[1].ID != LegacyDefaultInstanceID {
		t.Fatalf("first redis keeps id: %+v", spec.Instances[1])
	}
	if spec.Instances[2].ID == LegacyDefaultInstanceID {
		t.Fatalf("duplicate redis id reassigned: %+v", spec.Instances[2])
	}
}

func TestPublicS3Host_RustFS(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "db.example.com")
	defaultInst := ServiceInstance{ID: LegacyDefaultInstanceID, Type: "rustfs"}
	if got := PublicS3Host(defaultInst); got != "s3-default.db.example.com" {
		t.Fatalf("default host: %q", got)
	}
	hosts := KongRustFSHosts(defaultInst)
	if len(hosts) != 2 || hosts[0] != "s3-default.db.example.com" || hosts[1] != "s3.db.example.com" {
		t.Fatalf("%v", hosts)
	}
	auto := ServiceInstance{ID: "calm-otter", Type: "rustfs"}
	if got := PublicS3Host(auto); got != "s3-calm-otter.db.example.com" {
		t.Fatal(got)
	}
}

func TestPublicS3Host_IPUsesSSLip(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	inst := ServiceInstance{ID: "pqr", Type: "rustfs"}
	want := "s3-pqr.178-105-35-230.sslip.io"
	if got := PublicS3Host(inst); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := PublicS3Endpoint(inst); got != "http://"+want+":9080" {
		t.Fatalf("endpoint %q", got)
	}
}
