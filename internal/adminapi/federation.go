package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/internal/tailscale"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type federationPatchRequest struct {
	AdvertiseURL *string `json:"advertise_url,omitempty"`
}

func advertiseURLForHost(host, port string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return fmt.Sprintf("http://%s/api/v1/peers/advertise", host)
	}
	return fmt.Sprintf("http://%s:%s/api/v1/peers/advertise", host, port)
}

func gatewayAdvertiseEndpoint(spec obelisk.Spec) string {
	public, _ := gatewayAdvertiseEndpoints(spec)
	return public
}

func gatewayAdvertiseEndpoints(_ obelisk.Spec) (publicURL, tailnetURL string) {
	port := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT"))
	if port == "" {
		port = "9080"
	}
	pub := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST"))
	if pub == "" {
		pub = strings.TrimSpace(obelisk.PublicHost())
	}
	publicURL = advertiseURLForHost(pub, port)

	st := tailscale.GetStatus(context.Background())
	tailHost := ""
	if len(st.IPs) > 0 {
		tailHost = strings.TrimSpace(st.IPs[0])
	} else {
		tailHost = strings.TrimSuffix(strings.TrimSpace(st.DNSName), ".")
	}
	tailnetURL = advertiseURLForHost(tailHost, port)

	if publicURL == "" {
		publicURL = tailnetURL
		tailnetURL = ""
	} else if tailnetURL == publicURL {
		tailnetURL = ""
	}
	return publicURL, tailnetURL
}

func (s *Server) handleGetFederation(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := map[string]any{
		"role":              spec.Deployment.Role,
		"profile":           spec.Deployment.Profile,
		"tailscale_enabled": spec.Deployment.Tailscale.Enabled,
		"is_gateway":        obelisk.IsGateway(spec),
		"advertise_url":     strings.TrimSpace(spec.PeerGateway.AdvertiseURL),
		"tailnet_host":      peer.ResolveTailnetHost(),
		"auto_admit":        obelisk.PeerGatewayAutoAdmit(spec),
	}
	if obelisk.IsGateway(spec) {
		pub, tail := gatewayAdvertiseEndpoints(spec)
		out["gateway_advertise_endpoint"] = pub
		if tail != "" {
			out["gateway_advertise_endpoint_tailnet"] = tail
		}
		reg, err := peer.LoadRegistry(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reg.MarkStale(2 * time.Minute)
		peers := reg.ListPeers()
		out["peers"] = peers
	} else {
		st := peer.GetAdvertiseStatus()
		out["advertise_status"] = st
		if strings.TrimSpace(spec.PeerGateway.AdvertiseURL) == "" &&
			strings.TrimSpace(os.Getenv("OBELISK_GATEWAY_ADVERTISE_URL")) == "" {
			out["advertise_configured"] = false
		} else {
			out["advertise_configured"] = true
		}
	}
	writeJSON(w, out)
}

func (s *Server) handlePatchFederation(w http.ResponseWriter, r *http.Request) {
	var req federationPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if obelisk.IsGateway(spec) {
		s.mu.Unlock()
		http.Error(w, "gateway appliances do not advertise to a remote hub; use locals to register here", http.StatusBadRequest)
		return
	}
	if req.AdvertiseURL != nil {
		spec.PeerGateway.AdvertiseURL = strings.TrimSpace(*req.AdvertiseURL)
	}
	obelisk.NormalizeDeployment(&spec)
	if err := config.Validate(&spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(s.ConfigPath, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.mu.Unlock()
	writeJSON(w, map[string]any{
		"ok":            true,
		"advertise_url": strings.TrimSpace(spec.PeerGateway.AdvertiseURL),
	})
}

func (s *Server) handleFederationAdvertiseNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if obelisk.IsGateway(spec) {
		http.Error(w, "gateway does not advertise outward", http.StatusBadRequest)
		return
	}
	cctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	err = peer.PostAdvertise(cctx, cfg, spec)
	if err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{
			"error":   "advertise_failed",
			"message": err.Error(),
		})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "advertise_status": peer.GetAdvertiseStatus()})
}