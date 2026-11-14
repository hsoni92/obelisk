package obelisk

import "testing"

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
}