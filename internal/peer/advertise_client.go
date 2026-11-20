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
	"github.com/hsoni92/obelisk/internal/tailscale"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// ResolveTailnetHost is the dial address peers publish for mesh proxying.
// OBELISK_TAILNET_HOST wins; otherwise tailscaled status (Tailscale IP or MagicDNS).
func ResolveTailnetHost() string {
	if h := strings.TrimSpace(os.Getenv("OBELISK_TAILNET_HOST")); h != "" {
		return h
	}
	st := tailscale.GetStatus(context.Background())
	if len(st.IPs) > 0 {
		return strings.TrimSpace(st.IPs[0])
	}
	dns := strings.TrimSuffix(strings.TrimSpace(st.DNSName), ".")
	return dns
}

// BuildAdvertiseRequest constructs a catalog snapshot from local spec.
func BuildAdvertiseRequest(spec obelisk.Spec) AdvertiseRequest {
	peerID := strings.TrimSpace(os.Getenv("OBELISK_PEER_ID"))
	if peerID == "" {
		peerID = strings.TrimSpace(spec.Machine.ID)
	}
	if peerID == "" {
		peerID = "local"
	}
	req := AdvertiseRequest{
		PeerID:         peerID,
		MachineID:      strings.TrimSpace(spec.Machine.ID),
		TailnetHost:    ResolveTailnetHost(),
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
		return fmt.Errorf("no gateway advertise URL configured")
	}
	err := postAdvertiseOnce(ctx, configPath, spec, url)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "invalid peer token") {
		ClearLocalToken(configPath)
		if err2 := postAdvertiseOnce(ctx, configPath, spec, url); err2 == nil {
			return nil
		}
	}
	return err
}

func postAdvertiseOnce(ctx context.Context, configPath string, spec obelisk.Spec, url string) error {
	reqBody := BuildAdvertiseRequest(spec)
	if reqBody.TailnetHost == "" {
		err := fmt.Errorf("tailnet host unknown (join Tailscale or set OBELISK_TAILNET_HOST)")
		RecordAdvertiseAttempt(err)
		return err
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		RecordAdvertiseAttempt(err)
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		RecordAdvertiseAttempt(err)
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if tok := ReadLocalToken(configPath); tok != "" {
		httpReq.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		err = fmt.Errorf("advertise request to gateway: %w", err)
		RecordAdvertiseAttempt(err)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("advertise %s: %s", resp.Status, strings.TrimSpace(string(body)))
		RecordAdvertiseAttempt(err)
		return err
	}
	var ar AdvertiseResponse
	if json.Unmarshal(body, &ar) == nil && ar.Token != "" {
		if werr := WriteLocalToken(configPath, ar.Token); werr != nil {
			err := fmt.Errorf("save peer token: %w", werr)
			RecordAdvertiseAttempt(err)
			return err
		}
	}
	RecordAdvertiseAttempt(nil)
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
		_ = PostAdvertise(cctx, cfg, spec)
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