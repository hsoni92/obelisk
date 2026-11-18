package obelisk

import (
	"fmt"
	"os"
	"strings"
)

// DefaultAdminHTTPPort is the host-facing Kong HTTP port for the admin UI and /api/v1.
const DefaultAdminHTTPPort = 9080

// DefaultNativeEdgeTLSPort is the host-facing Kong TLS port for native data services
// (Redis/Postgres/MySQL/Mongo). It is deliberately not 443 so an Obelisk appliance
// never collides with an existing :443 listener on a shared host.
const DefaultNativeEdgeTLSPort = 9443

// ReservedHostPorts are ports Obelisk refuses to bind on the host. On a shared VPS the
// host's existing edge (e.g. an app's own Kong/nginx Ingress) already owns these.
var ReservedHostPorts = map[int]string{
	80:  "HTTP",
	443: "HTTPS",
}

// ReservedHostPortConflict reports a human-readable label when port is reserved.
func ReservedHostPortConflict(port int) (string, bool) {
	label, ok := ReservedHostPorts[port]
	return label, ok
}

// isReservedHostPort reports whether port must not be used by Obelisk.
func isReservedHostPort(port int) bool {
	_, ok := ReservedHostPorts[port]
	return ok
}

// AdminHTTPPort returns the public Kong HTTP port (connection hints and admin URL).
// Ports 80/443 are never returned; an override that collides falls back to the default.
func AdminHTTPPort() int {
	if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err == nil && n > 0 && !isReservedHostPort(n) {
			return n
		}
	}
	return DefaultAdminHTTPPort
}

func AdminHTTPPortString() string {
	return fmt.Sprintf("%d", AdminHTTPPort())
}

// NativeEdgeTLSPortNumber returns the public Kong TLS port for native data services.
// Ports 80/443 are never returned; an override that collides falls back to the default.
func NativeEdgeTLSPortNumber() int {
	for _, env := range []string{"OBELISK_TLS_PORT", "OBELISK_EDGE_TLS_PORT"} {
		if p := strings.TrimSpace(os.Getenv(env)); p != "" {
			var n int
			if _, err := fmt.Sscanf(p, "%d", &n); err == nil && n > 0 && !isReservedHostPort(n) {
				return n
			}
		}
	}
	return DefaultNativeEdgeTLSPort
}

// NativeEdgeTLSPort returns the public TLS port for native data services as a string.
func NativeEdgeTLSPort() string {
	return fmt.Sprintf("%d", NativeEdgeTLSPortNumber())
}
