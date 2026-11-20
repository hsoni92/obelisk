package obelisk

import (
	"fmt"
	"os"
	"strings"
)

// PeerBackhaulEnabled reports whether this local appliance publishes tailnet backhaul for federation.
func PeerBackhaulEnabled(spec Spec) bool {
	if IsGateway(spec) {
		return false
	}
	if u := strings.TrimSpace(spec.PeerGateway.AdvertiseURL); u != "" {
		return true
	}
	return strings.TrimSpace(os.Getenv("OBELISK_GATEWAY_ADVERTISE_URL")) != ""
}

// PublicEdgeHostWithDomain is PublicEdgeHost using an explicit base domain (gateway sslip.io zone).
func PublicEdgeHostWithDomain(inst ServiceInstance, baseDomain string) string {
	baseDomain = strings.TrimSpace(baseDomain)
	if inst.Type == "" || inst.ID == "" || baseDomain == "" {
		return ""
	}
	prefix := inst.Type
	if prefix == "rustfs" {
		prefix = "s3"
	}
	return fmt.Sprintf("%s-%s.%s", prefix, inst.ID, baseDomain)
}

// FederatedKongEdgeHosts returns gateway-facing Host/SNI names for a peer-owned instance.
func FederatedKongEdgeHosts(peerID string, inst ServiceInstance, gatewayBaseDomain string) []string {
	fed := inst
	fed.ID = FederatedInstanceID(peerID, inst.ID)
	return KongEdgeHostsWithDomain(fed, gatewayBaseDomain)
}

// KongEdgeHostsWithDomain lists edge hostnames for inst under baseDomain.
func KongEdgeHostsWithDomain(inst ServiceInstance, baseDomain string) []string {
	host := PublicEdgeHostWithDomain(inst, baseDomain)
	if host == "" {
		return nil
	}
	out := []string{host}
	if inst.ID == LegacyDefaultInstanceID {
		short := inst.Type
		if short == "rustfs" {
			short = "s3"
		}
		out = append(out, short+"."+strings.TrimSpace(baseDomain))
	}
	return out
}