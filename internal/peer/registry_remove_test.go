package peer

import (
	"path/filepath"
	"testing"
)

func TestRemovePeer(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	reg, err := LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.ApplyAdvertisement(AdvertiseRequest{
		PeerID: "gone", TailnetHost: "100.64.0.1",
		Instances: []AdvertisedInstance{{Type: "redis", ID: "a", Enabled: true}},
	}, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.RemovePeer("gone"); err != nil {
		t.Fatal(err)
	}
	if err := reg.RemovePeer("gone"); err == nil {
		t.Fatal("expected not found")
	}
	reg2, err := LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg2.ListPeers()) != 0 {
		t.Fatalf("peers %+v", reg2.ListPeers())
	}
}

func TestRemovePeerWithToken(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	reg, err := LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := reg.ApplyAdvertisement(AdvertiseRequest{
		PeerID: "home", TailnetHost: "100.64.0.1",
	}, "", true)
	if err != nil || resp.Token == "" {
		t.Fatal(resp, err)
	}
	if err := reg.RemovePeerWithToken("home", "wrong"); err == nil {
		t.Fatal("expected auth failure")
	}
	if err := reg.RemovePeerWithToken("home", resp.Token); err != nil {
		t.Fatal(err)
	}
}
