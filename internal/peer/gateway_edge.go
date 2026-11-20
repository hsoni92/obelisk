package peer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const gatewayEdgeFile = "peer.gateway_edge.json"

type gatewayEdgeDisk struct {
	EdgeBaseDomain string `json:"edge_base_domain,omitempty"`
}

// WriteGatewayEdgeDomain persists the gateway public edge zone for local backhaul TLS/SNI.
func WriteGatewayEdgeDomain(configPath, domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil
	}
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(gatewayEdgeDisk{EdgeBaseDomain: domain}, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, gatewayEdgeFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, gatewayEdgeFile))
}

// LoadGatewayEdgeDomain reads the learned gateway edge zone.
func LoadGatewayEdgeDomain(configPath string) string {
	path := filepath.Join(config.CredentialsDir(configPath), gatewayEdgeFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var disk gatewayEdgeDisk
	if json.Unmarshal(data, &disk) != nil {
		return ""
	}
	return strings.TrimSpace(disk.EdgeBaseDomain)
}

// BackhaulInfo configures tailnet listeners and gateway SNI on a local peer.
type BackhaulInfo struct {
	PeerID            string
	GatewayEdgeDomain string
}

// LocalBackhaul returns exposure settings when this appliance advertises to a gateway.
func LocalBackhaul(configPath string, spec obelisk.Spec) (BackhaulInfo, bool) {
	if !obelisk.PeerBackhaulEnabled(spec) {
		return BackhaulInfo{}, false
	}
	peerID := strings.TrimSpace(spec.Machine.ID)
	if peerID == "" {
		peerID = strings.TrimSpace(os.Getenv("OBELISK_PEER_ID"))
	}
	if peerID == "" {
		return BackhaulInfo{}, false
	}
	domain := strings.TrimSpace(spec.PeerGateway.GatewayEdgeHost)
	if domain == "" {
		domain = LoadGatewayEdgeDomain(configPath)
	}
	if domain == "" {
		return BackhaulInfo{}, false
	}
	return BackhaulInfo{PeerID: peerID, GatewayEdgeDomain: domain}, true
}
