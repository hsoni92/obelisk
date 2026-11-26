package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/history"
	"github.com/hsoni92/obelisk/internal/peer"
)

func writeConnectivityConfig(t *testing.T, role string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	body := "deployment:\n  role: " + role + "\n  profile: vps\n"
	if role == "gateway" {
		body += "  tailscale:\n    enabled: true\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConnectivityLocalHostHasNoPeerView(t *testing.T) {
	cfg := writeConnectivityConfig(t, "local")
	srv := &Server{ConfigPath: cfg}
	rr := httptest.NewRecorder()
	srv.handleConnectivity(rr, httptest.NewRequest(http.MethodGet, "/api/v1/connectivity", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var got connectivityResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.PeerViewEnabled || got.IsGateway || len(got.Peers) != 0 {
		t.Fatalf("local host must not expose a peer view: %+v", got)
	}
}

func TestConnectivityGatewayShowsPeerStateAndStrip(t *testing.T) {
	cfg := writeConnectivityConfig(t, "gateway")
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.ApplyAdvertisement(peer.AdvertiseRequest{
		PeerID:      "home",
		MachineID:   "box-1",
		TailnetHost: "100.64.0.2",
	}, "", true); err != nil {
		t.Fatal(err)
	}

	store := history.NewStore(config.HistoryPath(cfg))
	now := time.Now().UTC()
	// Gateway observed for the last hour, peer connected for that hour.
	if err := store.Append(history.Event{At: now.Add(-time.Hour), Kind: peer.KindGateway, Key: peer.KindGateway, State: peer.GatewayUp}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(history.Event{At: now.Add(-time.Hour), Kind: peer.KindPeer, Key: peer.PeerHistoryKey("home"), State: peer.StateConnected}); err != nil {
		t.Fatal(err)
	}
	srv := &Server{ConfigPath: cfg, historyStore: store}
	srv.historyOnce.Do(func() {})

	rr := httptest.NewRecorder()
	srv.handleConnectivity(rr, httptest.NewRequest(http.MethodGet, "/api/v1/connectivity", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var got connectivityResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.PeerViewEnabled || len(got.Peers) != 1 {
		t.Fatalf("expected one peer in gateway view: %+v", got)
	}
	p := got.Peers[0]
	if p.State != peer.StateConnected || !p.HeartbeatFresh {
		t.Fatalf("peer state: %+v", p)
	}
	if !p.DuplicateMachineID && p.MachineID != "box-1" {
		t.Fatalf("machine id not reported: %+v", p)
	}
	if len(p.Strip) != peer.StripBinsNum {
		t.Fatalf("strip bins: %d", len(p.Strip))
	}
	if p.Strip[0] != peer.StateNoData {
		t.Fatalf("bins before monitoring must be no_data, got %s", p.Strip[0])
	}
	if p.Strip[len(p.Strip)-1] != peer.StateConnected {
		t.Fatalf("latest bin should be connected, got %s", p.Strip[len(p.Strip)-1])
	}
	if p.ObservedSeconds < 3000 {
		t.Fatalf("observed time should be about an hour, got %ds", p.ObservedSeconds)
	}
}
