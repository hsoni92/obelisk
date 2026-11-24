package peer

import (
	"path/filepath"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestFederatedTargets_UsesBackhaulPorts(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	reg, err := LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.ApplyAdvertisement(AdvertiseRequest{
		PeerID:             "home",
		TailnetHost:        "100.64.0.1",
		NativeBackhaulPort: 9443,
		HTTPBackhaulPort:   9080,
		Instances: []AdvertisedInstance{
			{Type: "postgres", ID: "default", Enabled: true},
		},
	}, "", true)
	if err != nil {
		t.Fatal(err)
	}
	spec := obelisk.DefaultSpec("vps")
	spec.Deployment.Role = obelisk.DeploymentRoleGateway
	targets := reg.FederatedTargets(spec, 0)
	if len(targets) != 1 {
		t.Fatalf("targets %d", len(targets))
	}
	if targets[0].NativePort != 9443 {
		t.Fatalf("native port %d", targets[0].NativePort)
	}
}
