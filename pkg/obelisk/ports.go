package obelisk

import (
	"fmt"
	"os"
	"strings"
)

// DefaultAdminHTTPPort is the host-facing Kong HTTP port for the admin UI and /api/v1.
const DefaultAdminHTTPPort = 9080

// AdminHTTPPort returns the public Kong HTTP port (connection hints and admin URL).
func AdminHTTPPort() int {
	if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return DefaultAdminHTTPPort
}

func AdminHTTPPortString() string {
	return fmt.Sprintf("%d", AdminHTTPPort())
}