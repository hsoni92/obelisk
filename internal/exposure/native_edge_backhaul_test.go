package exposure

import (
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestRenderNativeEdgeNginx_PeerBackhaulSNI(t *testing.T) {
	spec := obelisk.DefaultSpec("local")
	spec.Deployment.Role = obelisk.DeploymentRoleLocal
	spec.Machine.ID = "obelisk-470f4a4845ec"
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "eager-stork", Type: "postgres", Enabled: true,
	}}
	backhaul := &peer.BackhaulInfo{
		PeerID:            "obelisk-470f4a4845ec",
		GatewayEdgeDomain: "178-105-35-230.sslip.io",
	}
	out := RenderNativeEdgeNginx(spec, "obelisk", nil, backhaul)
	want := "postgres-obelisk-470f4a4845ec-eager-stork.178-105-35-230.sslip.io obelisk-postgres-eager-stork.obelisk.svc.cluster.local:5432"
	if !strings.Contains(out, want) {
		t.Fatalf("missing backhaul map:\n%s", out)
	}
}