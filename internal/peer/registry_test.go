package peer

import (
	"path/filepath"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestApplyAdvertisementAutoAdmit(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	r, err := LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := r.ApplyAdvertisement(AdvertiseRequest{
		PeerID:      "home",
		TailnetHost: "100.64.0.1",
		Instances: []AdvertisedInstance{
			{Type: "postgres", ID: "default", Enabled: true},
		},
	}, "", true)
	if err != nil || !resp.Accepted || resp.Token == "" {
		t.Fatalf("admit: %+v err=%v", resp, err)
	}
	resp2, err := r.ApplyAdvertisement(AdvertiseRequest{
		PeerID:      "home",
		TailnetHost: "100.64.0.1",
		Instances: []AdvertisedInstance{
			{Type: "postgres", ID: "default", Enabled: true},
		},
	}, resp.Token, true)
	if err != nil || !resp2.Accepted {
		t.Fatalf("second advertise: %+v err=%v", resp2, err)
	}
	resp3, err := r.ApplyAdvertisement(AdvertiseRequest{
		PeerID:      "home",
		TailnetHost: "100.64.0.1",
		Instances: []AdvertisedInstance{
			{Type: "postgres", ID: "default", Enabled: true},
		},
	}, "", true)
	if err != nil || !resp3.Accepted || resp3.Token == "" {
		t.Fatalf("re-admit after lost token: %+v err=%v", resp3, err)
	}
	spec := obelisk.DefaultSpec("vps")
	spec.Deployment.Role = obelisk.DeploymentRoleGateway
	targets := r.FederatedTargets(spec, 0)
	if len(targets) != 1 {
		t.Fatalf("expected 1 federated target, got %d", len(targets))
	}
}