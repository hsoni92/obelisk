package adminapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/tailscale"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type tailscaleConnectRequest struct {
	AuthKey         string `json:"auth_key"`
	EnableInConfig  bool   `json:"enable_in_config"`
	GatewayRole     bool   `json:"gateway_role"`
}

type deploymentPatchRequest struct {
	Role             *string `json:"role,omitempty"`
	TailscaleEnabled *bool   `json:"tailscale_enabled,omitempty"`
}

func (s *Server) handleTailscaleStatus(w http.ResponseWriter, r *http.Request) {
	st := tailscale.GetStatus(r.Context())
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err == nil {
		resp := map[string]any{
			"status":             st,
			"desired_enabled":    spec.Deployment.Tailscale.Enabled,
			"deployment_role":    spec.Deployment.Role,
			"deployment_profile": spec.Deployment.Profile,
		}
		if obelisk.IsGateway(spec) {
			pub, tail := gatewayAdvertiseEndpoints(spec)
			if pub != "" {
				resp["gateway_advertise_endpoint"] = pub
			}
			if tail != "" {
				resp["gateway_advertise_endpoint_tailnet"] = tail
			}
		}
		writeJSON(w, resp)
		return
	}
	writeJSON(w, map[string]any{"status": st, "error": err.Error()})
}

func (s *Server) handleTailscaleConnect(w http.ResponseWriter, r *http.Request) {
	var req tailscaleConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stBefore := tailscale.GetStatus(r.Context())
	if !stBefore.Connected {
		if err := tailscale.Up(r.Context(), req.AuthKey); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	if req.EnableInConfig || req.GatewayRole {
		s.mu.Lock()
		spec, err := config.Load(cfg)
		if err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if req.EnableInConfig || req.GatewayRole {
			spec.Deployment.Tailscale.Enabled = true
		}
		if req.GatewayRole {
			spec.Deployment.Role = obelisk.DeploymentRoleGateway
		}
		obelisk.NormalizeDeployment(&spec)
		obelisk.NormalizePeerGateway(&spec)
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
		rec := s.reconciler(cfg)
		_, _ = rec.Reconcile(r.Context())
	}
	st := tailscale.GetStatus(r.Context())
	writeJSON(w, map[string]any{"ok": true, "status": st})
}

func (s *Server) handlePatchDeployment(w http.ResponseWriter, r *http.Request) {
	var req deploymentPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(config.EnsureReadablePath(s.ConfigPath))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.Role != nil {
		role := strings.ToLower(strings.TrimSpace(*req.Role))
		switch role {
		case obelisk.DeploymentRoleLocal, obelisk.DeploymentRoleGateway:
			spec.Deployment.Role = role
		default:
			http.Error(w, "role must be local or gateway", http.StatusBadRequest)
			return
		}
	}
	if req.TailscaleEnabled != nil {
		spec.Deployment.Tailscale.Enabled = *req.TailscaleEnabled
	}
	if spec.Deployment.Role == obelisk.DeploymentRoleGateway {
		spec.Deployment.Tailscale.Enabled = true
	}
	if spec.Deployment.Profile == "vps" && spec.Deployment.Role == obelisk.DeploymentRoleLocal {
		spec.Deployment.Tailscale.Enabled = false
	}
	obelisk.NormalizeDeployment(&spec)
	obelisk.NormalizePeerGateway(&spec)
	if err := config.Validate(&spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(s.ConfigPath, spec); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rec := s.reconciler(s.ConfigPath)
	_, _ = rec.Reconcile(r.Context())
	writeJSON(w, map[string]any{
		"profile": spec.Deployment.Profile,
		"role":    spec.Deployment.Role,
		"tailscale": spec.Deployment.Tailscale.Enabled,
	})
}