package obelisk

import "testing"

func TestAllocateInstanceID_Unique(t *testing.T) {
	spec := DefaultSpec("vps")
	ids := map[string]struct{}{}
	for i := 0; i < 20; i++ {
		id := AllocateInstanceID(&spec)
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
		t.Fatalf("first keeps legacy id: %+v", spec.Instances[0])
	}
	if spec.Instances[1].ID == LegacyDefaultInstanceID {
		t.Fatalf("second should be reassigned: %+v", spec.Instances[1])
	}
}

func TestPublicHTTPPaths_RustFS(t *testing.T) {
	defaultInst := ServiceInstance{ID: LegacyDefaultInstanceID, Type: "rustfs"}
	paths := PublicHTTPPaths(defaultInst)
	if len(paths) != 2 || paths[0] != "/s3/default" || paths[1] != "/s3" {
		t.Fatalf("%v", paths)
	}
	auto := ServiceInstance{ID: "ia1b2c3d4", Type: "rustfs"}
	if p := PublicHTTPPath(auto); p != "/s3/ia1b2c3d4" {
		t.Fatal(p)
	}
}