package mailgateway

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/mail"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// Server is the HTTP mail send gateway.
type Server struct {
	ConfigPath string

	mu            sync.RWMutex
	enabled       bool
	gatewayToken  string
	selector      *Selector
	idem          *idempotency
	lastAccountID string
}

func NewServer(configPath string) *Server {
	return &Server{
		ConfigPath: configPath,
		selector:   NewSelector(nil),
		idem:       newIdempotency(),
	}
}

func (s *Server) Reload() error {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		return err
	}
	creds, _, err := mail.LoadCredentials(cfg)
	if err != nil {
		return err
	}
	var pool []mailAccount
	for _, a := range spec.MailGateway.Accounts {
		if !a.Enabled {
			continue
		}
		secret := strings.TrimSpace(creds.Accounts[a.ID])
		if secret == "" || a.Address == "" {
			continue
		}
		pool = append(pool, mailAccount{ID: a.ID, Address: a.Address, Secret: secret})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = spec.MailGateway.Enabled && len(pool) > 0 && creds.GatewayToken != ""
	s.gatewayToken = creds.GatewayToken
	s.selector = NewSelector(pool)
	return nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	return mux
}

type Status struct {
	Enabled             bool   `json:"enabled"`
	Configured          bool   `json:"configured"`
	LastAccountID       string `json:"last_account_id,omitempty"`
	EnabledAccountCount int    `json:"enabled_account_count"`
}

func (s *Server) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Status{
		Enabled:             s.enabled,
		Configured:          s.gatewayToken != "",
		LastAccountID:       s.lastAccountID,
		EnabledAccountCount: s.selector.Len(),
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	s.mu.RLock()
	enabled := s.enabled
	token := s.gatewayToken
	s.mu.RUnlock()
	if !enabled {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "mail gateway disabled or not configured"})
		return
	}
	clientTok := bearerToken(r)
	if clientTok == "" || len(clientTok) != len(token) ||
		subtle.ConstantTimeCompare([]byte(clientTok), []byte(token)) != 1 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid gateway token"})
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/send":
		s.handleSend(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return
	}
	var req sendRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey == "" {
		idemKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	if idemKey != "" {
		if prev, ok := s.idem.get(idemKey); ok {
			writeJSON(w, http.StatusOK, sendResponse{
				AccountID: prev.accountID,
				From:      prev.from,
				Accepted:  true,
				Deduped:   true,
			})
			return
		}
	}
	to := req.firstTo()
	if to == "" || strings.TrimSpace(req.Subject) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "to and subject required"})
		return
	}
	text := strings.TrimSpace(req.Text)
	html := strings.TrimSpace(req.HTML)
	if text == "" && html == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text or html required"})
		return
	}

	s.mu.RLock()
	sel := s.selector
	s.mu.RUnlock()
	var candidates []mailAccount
	if strings.TrimSpace(req.AccountID) != "" {
		acc, ok := sel.ByID(strings.TrimSpace(req.AccountID))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown account_id"})
			return
		}
		candidates = []mailAccount{acc}
		if req.Failover {
			for _, a := range sel.RotateList() {
				if a.ID != acc.ID {
					candidates = append(candidates, a)
				}
			}
		}
	} else {
		candidates = sel.RotateList()
	}

	var lastErr error
	for _, acc := range candidates {
		err := mail.SendMessage(acc.Address, acc.Secret, to, req.Subject, text, html)
		if err != nil {
			lastErr = err
			if !retryableSMTP(err) {
				break
			}
			continue
		}
		s.mu.Lock()
		s.lastAccountID = acc.ID
		s.mu.Unlock()
		if idemKey != "" {
			s.idem.put(idemKey, acc.ID, acc.Address)
		}
		log.Printf("mail_send account_id=%s provider=%s outcome=ok recipient_domain=%s", acc.ID, obelisk.MailboxDomain(acc.Address), domainOf(to))
		writeJSON(w, http.StatusOK, sendResponse{
			AccountID: acc.ID,
			From:      acc.Address,
			Accepted:  true,
		})
		return
	}
	msg := "could not send mail"
	if lastErr != nil {
		msg = publicError(lastErr)
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": msg})
}

func domainOf(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) < 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

func retryableSMTP(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection") ||
		strings.Contains(msg, "auth") ||
		strings.Contains(msg, "535")
}

func publicError(err error) string {
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "auth") {
		return "mail server rejected credentials"
	}
	if strings.Contains(strings.ToLower(msg), "timeout") || strings.Contains(strings.ToLower(msg), "connection") {
		return "could not reach mail server"
	}
	return "could not send mail"
}
