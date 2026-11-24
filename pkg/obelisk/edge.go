package obelisk

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// VPSEdge exposes data services on per-instance hostnames via Kong (HTTP on the admin
// port, native TCP on NativeEdgeTLSPort with TLS). Neither port is 80 or 443 so an
// appliance never collides with a co-located web server.
func VPSEdge(spec Spec) bool {
	return strings.EqualFold(strings.TrimSpace(spec.Deployment.Profile), "vps") || IsGateway(spec)
}

// FederatedInstanceID namespaces a peer-owned instance on the gateway edge.
func FederatedInstanceID(peerID, instanceID string) string {
	peerID = strings.TrimSpace(peerID)
	instanceID = strings.TrimSpace(instanceID)
	if peerID == "" {
		return instanceID
	}
	if instanceID == "" {
		return peerID
	}
	return peerID + "-" + instanceID
}

// PublicEdgeBaseDomain is the DNS suffix for instance hostnames (sslip.io when PUBLIC_HOST is an IP).
func PublicEdgeBaseDomain() string {
	host := PublicHost()
	if ip := net.ParseIP(host); ip != nil {
		return strings.ReplaceAll(host, ".", "-") + ".sslip.io"
	}
	return host
}

// PublicEdgeHost returns the virtual hostname for an enabled instance (e.g. redis-calm-otter.example.com).
// RustFS keeps the s3- prefix for SDK compatibility.
func PublicEdgeHost(inst ServiceInstance) string {
	if inst.Type == "" || inst.ID == "" {
		return ""
	}
	prefix := inst.Type
	if prefix == "rustfs" {
		prefix = "s3"
	}
	return fmt.Sprintf("%s-%s.%s", prefix, inst.ID, PublicEdgeBaseDomain())
}

// PublicHTTPEdgeURL is the client base URL on the Kong HTTP admin port.
func PublicHTTPEdgeURL(inst ServiceInstance) string {
	host := PublicEdgeHost(inst)
	if host == "" {
		return ""
	}
	if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" {
		return fmt.Sprintf("http://%s:%s", host, p)
	}
	return fmt.Sprintf("http://%s:%s", host, AdminHTTPPortString())
}

// KongEdgeHosts lists Host/SNI values for an instance (includes legacy short names for id "default").
func KongEdgeHosts(inst ServiceInstance) []string {
	host := PublicEdgeHost(inst)
	if host == "" {
		return nil
	}
	out := []string{host}
	if inst.ID == LegacyDefaultInstanceID {
		short := inst.Type
		if short == "rustfs" {
			short = "s3"
		}
		out = append(out, short+"."+PublicEdgeBaseDomain())
	}
	return out
}

// NativeEdgeTypes are wire protocols exposed on NativeEdgeTLSPort with TLS + SNI (not HTTP).
var NativeEdgeTypes = map[string]int{
	"redis":    6379,
	"postgres": 5432,
	"mysql":    3306,
	"mongodb":  27017,
	"unimeter": 7001,
}

// HTTPEdgeTypes are exposed on the Kong HTTP admin port by hostname.
var HTTPEdgeTypes = map[string]int{
	"rustfs":      9000,
	"qdrant":      6333,
	"searxng":     8080,
	"redpanda":    8081,
	"bunqueue":    6790,
	"browserless": 3000,
}

func nativeEdgePort(typ string) (int, bool) {
	p, ok := NativeEdgeTypes[typ]
	return p, ok
}

func httpEdgePort(typ string) (int, bool) {
	p, ok := HTTPEdgeTypes[typ]
	return p, ok
}
