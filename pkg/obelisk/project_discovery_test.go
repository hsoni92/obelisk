package obelisk

import "testing"

func TestProjectDiscoveryKongInClusterURL(t *testing.T) {
	got := ProjectDiscoveryKongInClusterURL("prj_abc", "obelisk")
	want := "http://obelisk-kong.obelisk.svc.cluster.local:9080/discover/v1/projects/prj_abc?audience=in_cluster"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}