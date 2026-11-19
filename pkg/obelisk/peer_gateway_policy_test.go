package obelisk

import "testing"

func TestPeerGatewayAutoAdmitDefault(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Deployment.Role = DeploymentRoleGateway
	spec.Deployment.Tailscale.Enabled = true
	if !PeerGatewayAutoAdmit(spec) {
		t.Fatal("expected auto admit default true on gateway")
	}
	f := false
	spec.PeerGateway.AutoAdmit = &f
	if PeerGatewayAutoAdmit(spec) {
		t.Fatal("expected explicit false")
	}
}

func TestPeerGatewayTLSModeDefault(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Deployment.Role = DeploymentRoleGateway
	NormalizePeerGateway(&spec)
	if PeerGatewayTLSMode(spec) != PeerGatewayTLSPassthrough {
		t.Fatalf("tls mode %q", PeerGatewayTLSMode(spec))
	}
}