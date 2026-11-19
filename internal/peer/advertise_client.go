package peer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// BuildAdvertiseRequest constructs a catalog snapshot from local spec.
func BuildAdvertiseRequest(spec obelisk.Spec) AdvertiseRequest {
	peerID := strings.TrimSpace(os.Getenv("OBELISK_PEER_ID"))
	if peerID == "" {
		peerID = strings.TrimSpace(spec.Machine.ID)
	}
	if peerID == "" {
		peerID = "local"
	}
	host := strings.TrimSpace(os.Getenv("OBELISK_TAILNET_HOST"))
	req := AdvertiseRequest{
		PeerID:         peerID,
		MachineID:      strings.TrimSpace(spec.Machine.ID),
		TailnetHost:    host,
		DeploymentRole: strings.TrimSpace(spec.Deployment.Role),
	}
	for _, inst := range spec.Instances {
		req.Instances = append(req.Instances, AdvertisedInstance{
			Type:    inst.Type,
			ID:      inst.ID,
			Enabled: inst.Enabled,
		})
	}
	return req
}

// PostAdvertise sends the snapshot to the gateway.
func PostAdvertise(ctx context.Context, configPath string, spec obelisk.Spec) error {
	if obelisk.IsGateway(spec) {
		return nil
	}
	url := strings.TrimSpace(spec.PeerGateway.AdvertiseURL)
	if url == "" {
		url = strings.TrimSpace(os.Getenv("OBELISK_GATEWAY_ADVERTISE_URL"))
	}
	if url == "" {
		return nil
	}
	reqBody := BuildAdvertiseRequest(spec)
	if reqBody.TailnetHost == "" {
		return fmt.Errorf("OBELISK_TAILNET_HOST not set; cannot advertise")
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if tok := ReadLocalToken(configPath); tok != "" {
		httpReq.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("advertise %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var ar AdvertiseResponse
	if json.Unmarshal(body, &ar) == nil && ar.Token != "" {
		_ = WriteLocalToken(configPath, ar.Token)
	}
	return nil
}

// StartAdvertiser periodically posts catalog updates from a local appliance.
func StartAdvertiser(ctx context.Context, configPath string, interval time.Duration) {
	if interval <= 0 {
		interval = 45 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	run := func() {
		cfg := config.EnsureReadablePath(configPath)
		spec, err := config.Load(cfg)
		if err != nil || obelisk.IsGateway(spec) {
			return
		}
		if spec.PeerGateway.AdvertiseURL == "" && strings.TrimSpace(os.Getenv("OBELISK_GATEWAY_ADVERTISE_URL")) == "" {
			return
		}
		cctx, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		if err := PostAdvertise(cctx, cfg, spec); err != nil {
			// logged by caller if desired
			_ = err
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}