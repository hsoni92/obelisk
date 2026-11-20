package peer

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PeersAPIBase strips /advertise from the gateway advertise URL.
func PeersAPIBase(advertiseURL string) string {
	advertiseURL = strings.TrimSpace(advertiseURL)
	if advertiseURL == "" {
		return ""
	}
	return strings.TrimSuffix(advertiseURL, "/advertise")
}

// PeersResignURL is the peer-token resign endpoint on the gateway.
func PeersResignURL(advertiseURL string) string {
	base := PeersAPIBase(advertiseURL)
	if base == "" {
		return ""
	}
	return strings.TrimSuffix(base, "/") + "/resign"
}

// NotifyGatewayUnregister asks the gateway to drop this peer (best-effort).
func NotifyGatewayUnregister(ctx context.Context, advertiseURL, peerID, bearer string) error {
	peerID = strings.TrimSpace(peerID)
	target := PeersResignURL(advertiseURL)
	if target == "" || peerID == "" || strings.TrimSpace(bearer) == "" {
		return nil
	}
	body := fmt.Sprintf(`{"peer_id":%q}`, peerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gateway unregister: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gateway unregister: %s", resp.Status)
	}
	return nil
}

// ClearLocalRegistration removes local peer credentials used for federation.
func ClearLocalRegistration(configPath string) {
	ClearLocalToken(configPath)
	ClearGatewayEdgeDomain(configPath)
}