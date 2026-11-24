package obelisk

import "testing"

func TestFederatedKongEdgeHosts(t *testing.T) {
	inst := ServiceInstance{Type: "postgres", ID: "eager-stork", Enabled: true}
	got := FederatedKongEdgeHosts("obelisk-470f4a4845ec", inst, "178-105-35-230.sslip.io")
	want := "postgres-obelisk-470f4a4845ec-eager-stork.178-105-35-230.sslip.io"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %v want %q", got, want)
	}
}
