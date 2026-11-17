package obelisk

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// KongRouteServiceName is the declarative Kong service name for an instance route.
func KongRouteServiceName(inst ServiceInstance) string {
	return fmt.Sprintf("route-%s-%s", inst.Type, inst.ID)
}

// PublicHTTPPaths returns Kong path prefixes for HTTP-exposed instance types.
// RustFS uses host-based Kong routes (SigV4 cannot be path-prefixed); see PublicS3Host.
func PublicHTTPPaths(inst ServiceInstance) []string {
	switch inst.Type {
	case "rustfs":
		return nil
	default:
		return nil
	}
}

// PublicHTTPPath returns the canonical public HTTP path prefix for an instance, or "".
func PublicHTTPPath(inst ServiceInstance) string {
	paths := PublicHTTPPaths(inst)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// PublicS3BaseDomain is the DNS suffix shared by RustFS instance hostnames.
// Bare IPs map to sslip.io so clients can resolve s3-<id>.<ip-as-label>.sslip.io without extra DNS.
func PublicS3BaseDomain() string {
	host := PublicHost()
	if ip := net.ParseIP(host); ip != nil {
		return strings.ReplaceAll(host, ".", "-") + ".sslip.io"
	}
	return host
}

// PublicS3Host is the virtual host clients and SDKs should use for an instance's S3 API (root path /).
func PublicS3Host(inst ServiceInstance) string {
	if inst.Type != "rustfs" {
		return ""
	}
	return fmt.Sprintf("s3-%s.%s", inst.ID, PublicS3BaseDomain())
}

// PublicS3Endpoint is the full S3 base URL (no bucket suffix) for SDK endpoint / endpoint_url configuration.
func PublicS3Endpoint(inst ServiceInstance) string {
	host := PublicS3Host(inst)
	if host == "" {
		return ""
	}
	if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" {
		return fmt.Sprintf("http://%s:%s", host, p)
	}
	return "http://" + host
}

// KongRustFSHosts lists Host header values Kong should route to this RustFS instance.
func KongRustFSHosts(inst ServiceInstance) []string {
	if inst.Type != "rustfs" {
		return nil
	}
	hosts := []string{PublicS3Host(inst)}
	if inst.ID == LegacyDefaultInstanceID {
		hosts = append(hosts, "s3."+PublicS3BaseDomain())
	}
	return hosts
}

// RustFSServerDomains is the value for RUSTFS_SERVER_DOMAINS (Host header clients send, including port when non-default).
func RustFSServerDomains(inst ServiceInstance) string {
	host := PublicS3Host(inst)
	if host == "" {
		return ""
	}
	if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" && p != "80" {
		return host + ":" + p
	}
	return host
}