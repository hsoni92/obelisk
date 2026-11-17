package obelisk

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// NativeEdgeTLSPort is the single VPS port used for all native TCP services (multiplexed by SNI).
const NativeEdgeTLSPort = "443"

// VPSEdge exposes data services on per-instance hostnames via Kong (HTTP :80, native TCP :443 TLS).
func VPSEdge(spec Spec) bool {
	return strings.EqualFold(strings.TrimSpace(spec.Deployment.Profile), "vps")
}

// PublicEdgeBaseDomain is the DNS suffix for instance hostnames (sslip.io when PUBLIC_HOST is an IP).
func PublicEdgeBaseDomain() string {
	host := PublicHost()
	if ip := net.ParseIP(host); ip != nil {
		return strings.ReplaceAll(host, ".", "-") + ".sslip.io"
	}
	return host
}

// PublicEdgeHost returns the virtual hostname for an enabled instance (e.g. redis-pqr.example.com).
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

// PublicHTTPEdgeURL is the client base URL on Kong HTTP (port 80).
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

// NativeEdgeTypes are wire protocols exposed on :443 with TLS + SNI (not HTTP).
var NativeEdgeTypes = map[string]int{
	"redis":    6379,
	"postgres": 5432,
	"mysql":    3306,
	"mongodb":  27017,
}

// HTTPEdgeTypes are exposed on Kong HTTP :80 by hostname.
var HTTPEdgeTypes = map[string]int{
	"rustfs":  9000,
	"qdrant":  6333,
	"searxng": 8080,
}

func nativeEdgePort(typ string) (int, bool) {
	p, ok := NativeEdgeTypes[typ]
	return p, ok
}

func httpEdgePort(typ string) (int, bool) {
	p, ok := HTTPEdgeTypes[typ]
	return p, ok
}