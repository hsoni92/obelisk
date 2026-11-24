package obelisk

import "testing"

func TestRemapCanonicalDefaultIDs(t *testing.T) {
	spec := Spec{Instances: []ServiceInstance{
		{ID: "ib53fe6daa8f", Type: "redis", DisplayName: "default", Enabled: true},
		{ID: LegacyDefaultInstanceID, Type: "postgres", DisplayName: "default", Enabled: true},
	}}
	remapCanonicalDefaultIDs(&spec)
	if spec.Instances[0].ID != LegacyDefaultInstanceID {
		t.Fatalf("redis id = %q", spec.Instances[0].ID)
	}
}
