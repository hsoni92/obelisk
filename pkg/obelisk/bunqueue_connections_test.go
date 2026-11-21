package obelisk

import (
	"strings"
	"testing"
)

func TestBuildConnections_VPSBunqueueEdge(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "localhost")
	spec := DefaultSpec("vps")
	spec.Instances = []ServiceInstance{{
		ID: "moral-frog", Type: "bunqueue", Enabled: true,
	}}
	c := BuildConnections(spec, nil)
	if len(c.Instances) != 1 {
		t.Fatalf("%+v", c)
	}
	ic := c.Instances[0]
	wantTCP := "bunqueue-moral-frog.localhost:6789"
	if ic.BunqueueTCP != wantTCP {
		t.Fatalf("tcp %q want %q", ic.BunqueueTCP, wantTCP)
	}
	wantHealth := "http://bunqueue-moral-frog.localhost:9080/health"
	if ic.BunqueueHealthHTTP != wantHealth {
		t.Fatalf("health %q want %q", ic.BunqueueHealthHTTP, wantHealth)
	}
	use := instanceUse(spec, spec.Instances[0], ic)
	if !strings.Contains(use, "BUNQUEUE_ADDR="+wantTCP) || !strings.Contains(use, "BUNQUEUE_HEALTH_URL="+wantHealth) {
		t.Fatalf("use %q", use)
	}
}