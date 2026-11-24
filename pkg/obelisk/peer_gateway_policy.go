package obelisk

import "strings"

// Federation TLS mode (OQ1): v1 is SNI L4 passthrough only — no termination at gateway.
const PeerGatewayTLSPassthrough = "passthrough"

// PeerGatewayAutoAdmit returns whether unknown peers may auto-register (OQ2).
// Default true when role=gateway unless auto_admit is explicitly false.
func PeerGatewayAutoAdmit(spec Spec) bool {
	if !IsGateway(spec) {
		return false
	}
	if spec.PeerGateway.AutoAdmit != nil {
		return *spec.PeerGateway.AutoAdmit
	}
	return true
}

// PeerGatewayExposeNativePublic reports whether federated native TCP uses the gateway :9443 edge (OQ4).
func PeerGatewayExposeNativePublic(spec Spec) bool {
	if spec.PeerGateway.ExposeNativePublic != nil {
		return *spec.PeerGateway.ExposeNativePublic
	}
	return true
}

// PeerGatewayTLSMode returns the locked TLS handling for federation (OQ1).
func PeerGatewayTLSMode(spec Spec) string {
	if m := stringsTrim(spec.PeerGateway.TLSMode); m != "" {
		return m
	}
	return PeerGatewayTLSPassthrough
}

// NormalizePeerGateway applies defaults for gateway appliances.
func NormalizePeerGateway(spec *Spec) {
	if spec == nil || !IsGateway(*spec) {
		return
	}
	if spec.PeerGateway.TLSMode == "" {
		spec.PeerGateway.TLSMode = PeerGatewayTLSPassthrough
	}
}

func stringsTrim(s string) string { return strings.TrimSpace(s) }
