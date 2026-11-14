package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/mail"
	"github.com/hsoni92/obelisk/internal/mailgateway"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type mailAccountView struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Enabled    bool   `json:"enabled"`
	Address    string `json:"address"`
	Provider   string `json:"provider,omitempty"`
	HasSecret  bool   `json:"has_secret"`
	Notes      string `json:"notes,omitempty"`
}

type mailConfigResponse struct {
	Enabled             bool                `json:"enabled"`
	GatewayConfigured   bool                `json:"gateway_token_configured"`
	GatewayPublicPath   string              `json:"gateway_public_path"`
	GatewayListenHint   string              `json:"gateway_direct_path"`
	GatewayLocalBaseURL string              `json:"gateway_local_base_url"`
	Accounts            []mailAccountView   `json:"accounts"`
	Runtime             mailgateway.Status  `json:"runtime"`
}

type patchMailConfigRequest struct {
	Enabled                *bool `json:"enabled,omitempty"`
	RegenerateGatewayToken *bool `json:"regenerate_gateway_token,omitempty"`
}

type postMailAccountRequest struct {
	Label   string `json:"label"`
	Address string `json:"address"`
	Secret  string `json:"secret"`
	Enabled *bool  `json:"enabled,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

type patchMailAccountRequest struct {
	Label    *string `json:"label,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	Secret   *string `json:"secret,omitempty"`
	Address  *string `json:"address,omitempty"`
}

type mailTestRequest struct {
	GatewayURL string `json:"gateway_url,omitempty"`
	To         string `json:"to,omitempty"`
}

func (s *Server) buildMailConfig(cfg string, spec obelisk.Spec) mailConfigResponse {
	creds, _, _ := mail.LoadCredentials(cfg)
	var views []mailAccountView
	for _, a := range spec.MailGateway.Accounts {
		sec, has := creds.Accounts[a.ID]
		views = append(views, mailAccountView{
			ID:        a.ID,
			Label:     a.Label,
			Enabled:   a.Enabled,
			Address:   a.Address,
			Provider:  a.Provider,
			HasSecret: has && strings.TrimSpace(sec) != "",
			Notes:     a.Notes,
		})
	}
	rt := mailgateway.Status{}
	if s.MailGateway != nil {
		rt = s.MailGateway.Status()
	}
	localBase := gatewayLocalBaseURL(s.MailGatewayListenAddr)
	return mailConfigResponse{
		Enabled:             spec.MailGateway.Enabled,
		GatewayConfigured:   mail.GatewayConfigured(cfg),
		GatewayPublicPath:   "/mail/api/v1",
		GatewayListenHint:   "http://<host>" + localListenSuffix(s.MailGatewayListenAddr) + "/api/v1",
		GatewayLocalBaseURL: localBase,
		Accounts:            views,
		Runtime:             rt,
	}
}

func (s *Server) reloadMailGateway() {
	if s.MailGateway != nil {
		_ = s.MailGateway.Reload()
	}
}

func (s *Server) handleGetMailConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.buildMailConfig(cfg, spec))
}

func (s *Server) handlePatchMailConfig(w http.ResponseWriter, r *http.Request) {
	var req patchMailConfigRequest
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
		spec.MailGateway.Enabled = *req.Enabled
	}
	if req.RegenerateGatewayToken != nil && *req.RegenerateGatewayToken {
		tok, err := newGatewayToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		creds, _, err := mail.LoadCredentials(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		creds.GatewayToken = tok
		if err := mail.SaveCredentials(cfg, creds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"gateway_token": tok})
		s.reloadMailGateway()
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadMailGateway()
	writeJSON(w, s.buildMailConfig(cfg, spec))
}

func (s *Server) handlePostMailAccount(w http.ResponseWriter, r *http.Request) {
	var req postMailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	address := strings.ToLower(strings.TrimSpace(req.Address))
	secret := strings.TrimSpace(req.Secret)
	if address == "" || !strings.Contains(address, "@") {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}
	if secret == "" {
		http.Error(w, "secret required", http.StatusBadRequest)
		return
	}
	if _, err := mail.LookupSMTP(address); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if err := mail.ValidateSMTPAuth(address, secret); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		writeJSON(w, map[string]string{"error": publicMailErr(err)})
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = address
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id := fmt.Sprintf("mail-%d", time.Now().UnixNano())
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dup, ok := obelisk.FindMailAccountByAddress(spec.MailGateway.Accounts, address, ""); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":               "duplicate_address",
			"message":             fmt.Sprintf("address already in pool as %q", dup.Label),
			"existing_account_id": dup.ID,
		})
		return
	}
	acc := obelisk.MailAccount{
		ID: id, Label: label, Enabled: enabled, Address: address,
		Provider: obelisk.MailboxDomain(address), Notes: strings.TrimSpace(req.Notes),
	}
	spec.MailGateway.Accounts = append(spec.MailGateway.Accounts, acc)
	creds, _, err := mail.LoadCredentials(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	creds.Accounts[id] = secret
	if err := mail.SaveCredentials(cfg, creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadMailGateway()
	writeJSON(w, map[string]any{
		"account": mailAccountView{
			ID: id, Label: label, Enabled: enabled, Address: address,
			Provider: acc.Provider, HasSecret: true, Notes: acc.Notes,
		},
	})
}

func publicMailErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "auth") {
		return "mail server rejected credentials"
	}
	return msg
}

func (s *Server) handlePatchMailAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req patchMailAccountRequest
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
	for i, a := range spec.MailGateway.Accounts {
		if a.ID != id {
			continue
		}
		found = true
		if req.Label != nil {
			spec.MailGateway.Accounts[i].Label = strings.TrimSpace(*req.Label)
		}
		if req.Enabled != nil {
			spec.MailGateway.Accounts[i].Enabled = *req.Enabled
		}
		if req.Notes != nil {
			spec.MailGateway.Accounts[i].Notes = strings.TrimSpace(*req.Notes)
		}
		if req.Address != nil {
			addr := strings.ToLower(strings.TrimSpace(*req.Address))
			if addr != "" {
				if dup, ok := obelisk.FindMailAccountByAddress(spec.MailGateway.Accounts, addr, id); ok {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error":               "duplicate_address",
						"existing_account_id": dup.ID,
					})
					return
				}
				spec.MailGateway.Accounts[i].Address = addr
				spec.MailGateway.Accounts[i].Provider = obelisk.MailboxDomain(addr)
			}
		}
		break
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if req.Secret != nil && strings.TrimSpace(*req.Secret) != "" {
		secret := strings.TrimSpace(*req.Secret)
		for i, a := range spec.MailGateway.Accounts {
			if a.ID == id {
				if err := mail.ValidateSMTPAuth(a.Address, secret); err != nil {
					http.Error(w, publicMailErr(err), http.StatusUnprocessableEntity)
					return
				}
				creds, _, err := mail.LoadCredentials(cfg)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				creds.Accounts[id] = secret
				if err := mail.SaveCredentials(cfg, creds); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				spec.MailGateway.Accounts[i] = a
				break
			}
		}
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadMailGateway()
	writeJSON(w, s.buildMailConfig(cfg, spec))
}

func (s *Server) handleDeleteMailAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := spec.MailGateway.Accounts[:0]
	found := false
	for _, a := range spec.MailGateway.Accounts {
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
	spec.MailGateway.Accounts = out
	creds, _, err := mail.LoadCredentials(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	delete(creds.Accounts, id)
	if err := mail.SaveCredentials(cfg, creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.reloadMailGateway()
	writeJSON(w, map[string]string{"deleted": id})
}

func (s *Server) handleMailTest(w http.ResponseWriter, r *http.Request) {
	var req mailTestRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	creds, _, err := mail.LoadCredentials(cfg)
	if err != nil || creds.GatewayToken == "" {
		http.Error(w, "gateway token not configured", http.StatusBadRequest)
		return
	}
	var fromAcc *obelisk.MailAccount
	for _, a := range spec.MailGateway.Accounts {
		if a.Enabled && strings.TrimSpace(creds.Accounts[a.ID]) != "" {
			ac := a
			fromAcc = &ac
			break
		}
	}
	if fromAcc == nil {
		http.Error(w, "no enabled mailbox with secret", http.StatusBadRequest)
		return
	}
	to := strings.TrimSpace(req.To)
	if to == "" {
		to = fromAcc.Address
	}
	secret := creds.Accounts[fromAcc.ID]
	if err := mail.SendMessage(fromAcc.Address, secret, to, "Obelisk mail gateway test", "Test message from Obelisk mail gateway.", ""); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": publicMailErr(err)})
		return
	}
	writeJSON(w, map[string]any{
		"ok":         true,
		"message":    "test send accepted by SMTP server",
		"from":       fromAcc.Address,
		"account_id": fromAcc.ID,
		"to":         to,
	})
}