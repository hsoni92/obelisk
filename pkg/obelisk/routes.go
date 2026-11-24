package obelisk

import (
	"fmt"
	"os"
	"strings"
)

// KongRouteServiceName is the declarative Kong service name for an instance route.
func KongRouteServiceName(inst ServiceInstance) string {
	return fmt.Sprintf("route-%s-%s", inst.Type, inst.ID)
}

// KongTCPRouteServiceName is the Kong stream service name for native TLS edge routes.
func KongTCPRouteServiceName(inst ServiceInstance) string {
	return fmt.Sprintf("route-tcp-%s-%s", inst.Type, inst.ID)
}

// PublicHTTPPaths returns Kong path prefixes for HTTP-exposed instance types.
// VPS uses hostname-based Kong routes on the Kong HTTP port instead of path prefixes.
func PublicHTTPPaths(inst ServiceInstance) []string {
	return nil
}

// PublicHTTPPath returns the canonical public HTTP path prefix for an instance, or "".
func PublicHTTPPath(inst ServiceInstance) string {
	paths := PublicHTTPPaths(inst)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// PublicS3Host is the RustFS S3 virtual host (s3- prefix).
func PublicS3Host(inst ServiceInstance) string {
	if inst.Type != "rustfs" {
		return ""
	}
	return PublicEdgeHost(inst)
}

// PublicS3Endpoint is the full S3 base URL for SDKs (Kong HTTP port).
func PublicS3Endpoint(inst ServiceInstance) string {
	if inst.Type != "rustfs" {
		return ""
	}
	return PublicHTTPEdgeURL(inst)
}

// KongRustFSHosts lists Host header values Kong should route to this RustFS instance.
func KongRustFSHosts(inst ServiceInstance) []string {
	if inst.Type != "rustfs" {
		return nil
	}
	return KongEdgeHosts(inst)
}

// RustFSServerDomains is RUSTFS_SERVER_DOMAINS (client Host header, with non-default HTTP port if set).
func RustFSServerDomains(inst ServiceInstance) string {
	host := PublicS3Host(inst)
	if host == "" {
		return ""
	}
	if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" {
		return host + ":" + p
	}
	if AdminHTTPPort() != 80 {
		return host + ":" + AdminHTTPPortString()
	}
	return host
}
