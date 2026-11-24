package openroutergateway

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/openrouter"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const openRouterBase = "https://openrouter.ai"

// Server is the OpenRouter-compatible reverse proxy (separate listen port from admin API).
type Server struct {
	ConfigPath string

	mu            sync.RWMutex
	enabled       bool
	gatewayToken  string
	selector      *Selector
	client        *http.Client
	lastAccountID string
}

func NewServer(configPath string) *Server {
	return &Server{
		ConfigPath: configPath,
		client: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ResponseHeaderTimeout: 120 * time.Second,
			},
		},
		selector: NewSelector(nil),
	}
}

// Reload reads obelisk.yaml and credentials from disk.
func (s *Server) Reload() error {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		return err
	}
	creds, _, err := openrouter.LoadCredentials(cfg)
	if err != nil {
		return err
	}
	var upstream []upstreamAccount
	for _, a := range spec.OpenRouterGateway.Accounts {
		if !a.Enabled {
			continue
		}
		key := strings.TrimSpace(creds.Accounts[a.ID])
		if key == "" {
			continue
		}
		upstream = append(upstream, upstreamAccount{ID: a.ID, Key: key})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = spec.OpenRouterGateway.Enabled && len(upstream) > 0 && creds.GatewayToken != ""
	s.gatewayToken = creds.GatewayToken
	s.selector = NewSelector(upstream)
	return nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	return mux
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
	urlPath := r.URL.Path
	if !strings.HasPrefix(urlPath, "/api/v1/") {
		writeGatewayJSON(w, http.StatusNotFound, "not found")
		return
	}
	if !pathAllowed(urlPath) {
		writeGatewayJSON(w, http.StatusNotFound, "path not supported in gateway v1")
		return
	}

	s.mu.RLock()
	enabled := s.enabled
	token := s.gatewayToken
	accounts := s.selector.Snapshot()
	s.mu.RUnlock()

	if !enabled {
		writeGatewayJSON(w, http.StatusServiceUnavailable, "openrouter gateway disabled or not configured")
		return
	}
	clientTok := bearerToken(r)
	if clientTok == "" || len(clientTok) != len(token) ||
		subtle.ConstantTimeCompare([]byte(clientTok), []byte(token)) != 1 {
		writeGatewayJSON(w, http.StatusUnauthorized, "invalid gateway token")
		return
	}
	if len(accounts) == 0 {
		writeGatewayJSON(w, http.StatusServiceUnavailable, "no enabled openrouter accounts")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeGatewayJSON(w, http.StatusBadRequest, "read body")
		return
	}
	_ = r.Body.Close()
	streaming := bodyStreams(body)

	var lastStatus int
	for i, acc := range accounts {
		resp, err := s.doUpstream(r, body, acc, urlPath)
		if err != nil {
			if i == len(accounts)-1 {
				writeGatewayJSON(w, http.StatusBadGateway, err.Error())
				return
			}
			continue
		}
		s.mu.Lock()
		s.lastAccountID = acc.ID
		s.mu.Unlock()

		if !retryUpstreamStatus(resp.StatusCode) || i == len(accounts)-1 || streaming {
			forwardResponse(w, resp)
			return
		}
		lastStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	writeGatewayJSON(w, lastStatus, "upstream failed for all accounts")
}

// Status returns a snapshot for admin API.
type Status struct {
	Enabled       bool   `json:"enabled"`
	Configured    bool   `json:"configured"`
	LastAccountID string `json:"last_account_id,omitempty"`
	AccountCount  int    `json:"enabled_account_count"`
}

func (s *Server) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Status{
		Enabled:       s.enabled,
		Configured:    s.gatewayToken != "",
		LastAccountID: s.lastAccountID,
		AccountCount:  s.selector.Len(),
	}
}

// PingModels calls upstream GET /api/v1/models using the gateway token against local gateway addr.
func PingModels(gatewayHTTP string, gatewayToken string) (int, string, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(gatewayHTTP, "/")+"/api/v1/models", nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+gatewayToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(data), nil
}

// BuildSelectorFromSpec is used in tests.
func BuildSelectorFromSpec(spec obelisk.Spec, creds openrouter.CredentialsFile) *Selector {
	var upstream []upstreamAccount
	for _, a := range spec.OpenRouterGateway.Accounts {
		if !a.Enabled {
			continue
		}
		if key := strings.TrimSpace(creds.Accounts[a.ID]); key != "" {
			upstream = append(upstream, upstreamAccount{ID: a.ID, Key: key})
		}
	}
	return NewSelector(upstream)
}

// JSONError for tests.
func JSONError(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}
