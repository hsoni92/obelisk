package adminapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/openrouter"
	"github.com/hsoni92/obelisk/internal/openroutergateway"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type openRouterAccountView struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	Enabled         bool   `json:"enabled"`
	Notes           string `json:"notes,omitempty"`
	HasAPIKey       bool   `json:"has_api_key"`
	UpstreamLabel   string `json:"upstream_label,omitempty"`
	KeyTail         string `json:"key_tail,omitempty"`
	UpstreamUserID  string `json:"upstream_user_id,omitempty"`
	KeyFingerprint  string `json:"key_fingerprint,omitempty"`
}

type openRouterConfigResponse struct {
	Enabled              bool                    `json:"enabled"`
	GatewayConfigured    bool                    `json:"gateway_token_configured"`
	GatewayPublicPath    string                  `json:"gateway_public_path"`
	GatewayListenHint    string                  `json:"gateway_direct_path"`
	GatewayLocalBaseURL  string                  `json:"gateway_local_base_url"`
	Accounts             []openRouterAccountView `json:"accounts"`
	Runtime              openroutergateway.Status `json:"runtime"`
}

type patchOpenRouterConfigRequest struct {
	Enabled                *bool `json:"enabled,omitempty"`
	RegenerateGatewayToken *bool `json:"regenerate_gateway_token,omitempty"`
}

type postOpenRouterAccountRequest struct {
	Label   string `json:"label"`
	APIKey  string `json:"api_key"`
	Enabled *bool  `json:"enabled,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

type patchOpenRouterAccountRequest struct {
	Label    *string `json:"label,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	APIKey   *string `json:"api_key,omitempty"`
}

func (s *Server) buildOpenRouterConfig(cfg string, spec obelisk.Spec) openRouterConfigResponse {
	creds, _, _ := openrouter.LoadCredentials(cfg)
	var views []openRouterAccountView
	for _, a := range spec.OpenRouterGateway.Accounts {
		_, has := creds.Accounts[a.ID]
		key := strings.TrimSpace(creds.Accounts[a.ID])
		views = append(views, openRouterAccountView{
			ID:             a.ID,
			Label:          a.Label,
			Enabled:        a.Enabled,
			Notes:          a.Notes,
			HasAPIKey:      has && key != "",
			UpstreamLabel:  a.UpstreamLabel,
			KeyTail:        a.KeyTail,
			UpstreamUserID: a.UpstreamUserID,
			KeyFingerprint: a.KeyFingerprint,
		})
	}
	rt := openroutergateway.Status{}
	if s.Gateway != nil {
		rt = s.Gateway.Status()
	}
	localBase := gatewayLocalBaseURL(s.GatewayListenAddr)
	return openRouterConfigResponse{
		Enabled:           spec.OpenRouterGateway.Enabled,
		GatewayConfigured: openrouter.GatewayConfigured(cfg),
		GatewayPublicPath: "/openrouter/api/v1",
		GatewayListenHint: "http://<host>" + localListenSuffix(s.GatewayListenAddr) + "/api/v1",
		GatewayLocalBaseURL: localBase,
		Accounts:          views,
		Runtime:           rt,
	}
}

func localListenSuffix(addr string) string {
	if addr == "" {
		return ":8081"
	}
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	if host, port, err := net.SplitHostPort(addr); err == nil && host != "" {
		return ":" + port
	}
	return addr
}

func gatewayLocalBaseURL(addr string) string {
	if addr == "" {
		addr = ":8081"
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		if host == "" || host == "0.0.0.0" || host == "::" {
			return "http://127.0.0.1:" + port
		}
		return "http://" + net.JoinHostPort(host, port)
	}
	return "http://" + strings.TrimRight(addr, "/")
}

func (s *Server) reloadGateway() {
	if s.Gateway != nil {
		_ = s.Gateway.Reload()
	}
}

func (s *Server) handleGetOpenRouterConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.buildOpenRouterConfig(cfg, spec))
}

func (s *Server) handlePatchOpenRouterConfig(w http.ResponseWriter, r *http.Request) {
	var req patchOpenRouterConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.Enabled != nil {
		spec.OpenRouterGateway.Enabled = *req.Enabled
	}
	if req.RegenerateGatewayToken != nil && *req.RegenerateGatewayToken {
		tok, err := newGatewayToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		creds, _, err := openrouter.LoadCredentials(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		creds.GatewayToken = tok
		if err := openrouter.SaveCredentials(cfg, creds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"gateway_token": tok})
		s.reloadGateway()
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadGateway()
	writeJSON(w, s.buildOpenRouterConfig(cfg, spec))
}

func newGatewayToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "obk_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Server) handlePostOpenRouterAccount(w http.ResponseWriter, r *http.Request) {
	var req postOpenRouterAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		http.Error(w, "api_key required", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "account"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	meta, err := openrouter.LookupKey(apiKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := fmt.Sprintf("or-%d", time.Now().UnixNano())
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dup, ok := obelisk.FindAccountByFingerprint(spec.OpenRouterGateway.Accounts, meta.Fingerprint, ""); ok {
		writeDuplicateKey(w, dup, meta)
		return
	}
	acc := obelisk.OpenRouterAccount{
		ID: id, Label: label, Enabled: enabled, Notes: strings.TrimSpace(req.Notes),
	}
	applyKeyMetadata(&acc, meta)
	spec.OpenRouterGateway.Accounts = append(spec.OpenRouterGateway.Accounts, acc)
	creds, _, err := openrouter.LoadCredentials(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	creds.Accounts[id] = apiKey
	if err := openrouter.SaveCredentials(cfg, creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadGateway()
	writeJSON(w, map[string]any{
		"account": openRouterAccountView{
			ID:             id,
			Label:          label,
			Enabled:        enabled,
			HasAPIKey:      true,
			UpstreamLabel:  acc.UpstreamLabel,
			KeyTail:        acc.KeyTail,
			UpstreamUserID: acc.UpstreamUserID,
			KeyFingerprint: acc.KeyFingerprint,
		},
	})
}

func applyKeyMetadata(acc *obelisk.OpenRouterAccount, meta openrouter.KeyMetadata) {
	acc.KeyFingerprint = meta.Fingerprint
	acc.UpstreamLabel = meta.UpstreamLabel
	acc.KeyTail = meta.KeyTail
	acc.UpstreamUserID = meta.CreatorUserID
}

func writeDuplicateKey(w http.ResponseWriter, existing obelisk.OpenRouterAccount, meta openrouter.KeyMetadata) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":               "duplicate_key",
		"message":             fmt.Sprintf("this OpenRouter key is already in the pool as %q", existing.Label),
		"existing_account_id": existing.ID,
		"existing_label":      existing.Label,
		"upstream_label":      meta.UpstreamLabel,
		"key_tail":            meta.KeyTail,
	})
}

func (s *Server) handlePatchOpenRouterAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	var req patchOpenRouterAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	found := false
	for i, a := range spec.OpenRouterGateway.Accounts {
		if a.ID != id {
			continue
		}
		found = true
		if req.Label != nil {
			spec.OpenRouterGateway.Accounts[i].Label = strings.TrimSpace(*req.Label)
		}
		if req.Enabled != nil {
			spec.OpenRouterGateway.Accounts[i].Enabled = *req.Enabled
		}
		if req.Notes != nil {
			spec.OpenRouterGateway.Accounts[i].Notes = strings.TrimSpace(*req.Notes)
		}
		break
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if req.APIKey != nil && strings.TrimSpace(*req.APIKey) != "" {
		newKey := strings.TrimSpace(*req.APIKey)
		meta, err := openrouter.LookupKey(newKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if dup, ok := obelisk.FindAccountByFingerprint(spec.OpenRouterGateway.Accounts, meta.Fingerprint, id); ok {
			writeDuplicateKey(w, dup, meta)
			return
		}
		creds, _, err := openrouter.LoadCredentials(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		creds.Accounts[id] = newKey
		if err := openrouter.SaveCredentials(cfg, creds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for i, a := range spec.OpenRouterGateway.Accounts {
			if a.ID == id {
				applyKeyMetadata(&spec.OpenRouterGateway.Accounts[i], meta)
				break
			}
		}
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadGateway()
	writeJSON(w, s.buildOpenRouterConfig(cfg, spec))
}

func (s *Server) handleDeleteOpenRouterAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := spec.OpenRouterGateway.Accounts[:0]
	found := false
	for _, a := range spec.OpenRouterGateway.Accounts {
		if a.ID == id {
			found = true
			continue
		}
		out = append(out, a)
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	spec.OpenRouterGateway.Accounts = out
	creds, _, err := openrouter.LoadCredentials(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	delete(creds.Accounts, id)
	if err := openrouter.SaveCredentials(cfg, creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadGateway()
	writeJSON(w, map[string]string{"deleted": id})
}

func (s *Server) handleOpenRouterTest(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	creds, _, err := openrouter.LoadCredentials(cfg)
	if err != nil || creds.GatewayToken == "" {
		http.Error(w, "gateway token not configured", http.StatusBadRequest)
		return
	}
	gatewayURL := r.URL.Query().Get("gateway_url")
	if gatewayURL == "" {
		gatewayURL = gatewayLocalBaseURL(s.GatewayListenAddr)
	}
	code, body, err := openroutergateway.PingModels(gatewayURL, creds.GatewayToken)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ok := code >= 200 && code < 300
	preview := truncate(body, 512)
	msg := ""
	if !ok && strings.Contains(preview, `"error"`) {
		var envelope struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(preview), &envelope) == nil && envelope.Error != "" {
			msg = envelope.Error
		}
	}
	writeJSON(w, map[string]any{
		"ok":           ok,
		"status_code":  code,
		"body_preview": preview,
		"message":      msg,
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}