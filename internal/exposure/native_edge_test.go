package exposure

import (
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestRenderNativeEdgeNginx_RedisSNI(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "pqr", Type: "redis", Enabled: true,
	}}
	out := RenderNativeEdgeNginx(spec, "obelisk", nil)
	if !strings.Contains(out, "ssl_preread on;") {
		t.Fatalf("missing ssl_preread:\n%s", out)
	}
	if !strings.Contains(out, "redis-pqr.178-105-35-230.sslip.io obelisk-redis-pqr.obelisk.svc.cluster.local:6379") {
		t.Fatalf("missing redis map entry:\n%s", out)
	}
}

func TestRenderNativeEdgeNginx_SkipsDisabled(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "vps.example.com")
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "off", Type: "redis", Enabled: false,
	}}
	out := RenderNativeEdgeNginx(spec, "obelisk", nil)
	if strings.Contains(out, "redis-off.") {
		t.Fatalf("disabled instance in map:\n%s", out)
	}
}