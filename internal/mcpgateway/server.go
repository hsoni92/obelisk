// Package mcpgateway serves the built-in MCP tools over stateless streamable HTTP.
// It reaches its backends (SearXNG, mail) through narrow interfaces and imports
// no Kubernetes or controller code, so it can move to its own pod later.
package mcpgateway

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const (
	maxRequestBytes = 64 << 10
	serverName      = "obelisk"
	// Overrides the in-cluster SearXNG address (local dev uses a port-forward).
	searxURLEnv = "OBELISK_MCP_SEARXNG_URL"
)

// Versions this server speaks, newest first. Batching is rejected (removed in 2025-06-18).
var supportedVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

// Server is the MCP gateway.
type Server struct {
	ConfigPath string

	mailer  Mailer
	search  *searchBackend
	sendlog *sendLog
	tools   []Tool

	mu         sync.RWMutex
	enabled    bool
	tokens     []string
	toggles    map[string]bool
	searxURL   string
	calls      int
	lastCall   time.Time
	lastError  string
	authFails  int
	lastAuthAt time.Time
	lastAuthLg time.Time
	loadedAt   time.Time
}

// NewServer builds the gateway. mailer may be nil (send_mail then reports mail as missing).
func NewServer(configPath string, mailer Mailer) *Server {
	s := &Server{
		ConfigPath: configPath,
		mailer:     mailer,
		search:     newSearchBackend(),
		sendlog:    newSendLog(configPath),
	}
	s.tools = s.buildTools()
	return s
}

// Reload re-reads the spec and token file. The gateway stays disabled when no token exists.
func (s *Server) Reload() error {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		return err
	}
	creds, _, err := LoadCredentials(cfg)
	if err != nil {
		return err
	}
	var toks []string
	for _, t := range creds.Tokens {
		if strings.TrimSpace(t.Value) != "" {
			toks = append(toks, t.Value)
		}
	}
	searx := strings.TrimSpace(os.Getenv(searxURLEnv))
	if searx == "" {
		for _, inst := range spec.Instances {
			if inst.Type == "searxng" && inst.Enabled {
				searx = "http://" + obelisk.ClusterServiceFQDN(inst, "") + ":8080"
				break
			}
		}
	}
	toggles := make(map[string]bool, len(spec.MCPGateway.Tools))
	for k, v := range spec.MCPGateway.Tools {
		toggles[k] = v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = spec.MCPGateway.Enabled && len(toks) > 0
	s.tokens = toks
	s.toggles = toggles
	s.searxURL = searx
	s.loadedAt = time.Now()
	return nil
}

// maybeReload picks up hand-edited config, restored tokens and SearXNG instance changes
// without a controller restart. It does nothing until Reload has run once.
func (s *Server) maybeReload() {
	s.mu.RLock()
	stale := !s.loadedAt.IsZero() && time.Since(s.loadedAt) > 10*time.Second
	s.mu.RUnlock()
	if stale {
		_ = s.Reload()
	}
}

func (s *Server) searxBase() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searxURL
}

func (s *Server) toolEnabled(t Tool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.toggles[t.Name]; ok {
		return v
	}
	return t.DefaultEnabled
}

// ToolStatus is one tool as shown in the admin panel.
type ToolStatus struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Ready       bool   `json:"ready"`
	Message     string `json:"message,omitempty"`
	Destructive bool   `json:"destructive,omitempty"`
}

// Status is the runtime view for the admin API.
type Status struct {
	Enabled      bool         `json:"enabled"`
	Configured   bool         `json:"configured"`
	Tools        []ToolStatus `json:"tools"`
	Calls        int          `json:"calls"`
	LastCall     string       `json:"last_call,omitempty"`
	LastError    string       `json:"last_error,omitempty"`
	AuthFailures int          `json:"auth_failures"`
	LastAuthFail string       `json:"last_auth_failure,omitempty"`
	MailPerHour  int          `json:"mail_limit_per_hour"`
	MailPerDay   int          `json:"mail_limit_per_day"`
}

func (s *Server) Status() Status {
	st := Status{MailPerHour: maxMailPerHour, MailPerDay: maxMailPerDay}
	s.mu.RLock()
	st.Enabled = s.enabled
	st.Configured = len(s.tokens) > 0
	st.Calls = s.calls
	st.LastError = s.lastError
	st.AuthFailures = s.authFails
	if !s.lastCall.IsZero() {
		st.LastCall = s.lastCall.UTC().Format(time.RFC3339)
	}
	if !s.lastAuthAt.IsZero() {
		st.LastAuthFail = s.lastAuthAt.UTC().Format(time.RFC3339)
	}
	s.mu.RUnlock()
	for _, t := range s.tools {
		ready, msg := t.Backend()
		st.Tools = append(st.Tools, ToolStatus{
			Name: t.Name, Description: t.Description, Enabled: s.toolEnabled(t),
			Ready: ready, Message: msg, Destructive: t.Destructive,
		})
	}
	return st
}

// ToolNames lists registered tools, for validating admin toggles.
func (s *Server) ToolNames() []string {
	names := make([]string, 0, len(s.tools))
	for _, t := range s.tools {
		names = append(names, t.Name)
	}
	return names
}

// SendLog returns recent send_mail attempts, newest first.
func (s *Server) SendLog(limit int) []SendLogEntry { return s.sendlog.recent(limit) }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	return mux
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < 7 || !strings.EqualFold(h[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

func (s *Server) authorized(r *http.Request) bool {
	s.mu.RLock()
	enabled, tokens := s.enabled, s.tokens
	s.mu.RUnlock()
	got := bearerToken(r)
	if !enabled || got == "" {
		return false
	}
	ok := false
	for _, t := range tokens {
		if len(got) == len(t) && subtle.ConstantTimeCompare([]byte(got), []byte(t)) == 1 {
			ok = true
		}
	}
	return ok
}

func (s *Server) noteAuthFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authFails++
	s.lastAuthAt = time.Now()
	if time.Since(s.lastAuthLg) > 30*time.Second {
		s.lastAuthLg = time.Now()
		log.Printf("mcp_auth_fail total=%d", s.authFails)
	}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if p := strings.TrimRight(r.URL.Path, "/"); p != "" && p != "/mcp" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.maybeReload()
	// 401 for every failure, including "disabled", so the endpoint state is not revealed.
	if !s.authorized(r) {
		s.noteAuthFailure()
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// A browser page must not drive this endpoint; native clients send no Origin.
	if o := r.Header.Get("Origin"); o != "" {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	if err != nil {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32600, "batch requests are not supported"}})
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(trimmed, &req); err != nil || req.Method == "" {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "invalid JSON-RPC request"}})
		return
	}
	if len(req.ID) == 0 { // notification
		w.WriteHeader(http.StatusAccepted)
		return
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = s.initialize(req.Params)
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": s.listTools()}
	case "tools/call":
		resp.Result, resp.Error = s.callTool(r.Context(), req.Params)
	default:
		resp.Error = &rpcError{-32601, "method not found"}
	}
	writeRPC(w, resp)
}

func writeRPC(w http.ResponseWriter, v rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) initialize(params json.RawMessage) map[string]any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	version := supportedVersions[0]
	for _, v := range supportedVersions {
		if v == p.ProtocolVersion {
			version = v
		}
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": serverName, "version": "1"},
	}
}

// listTools always lists every tool, whether or not it is enabled or its backend is ready.
// A tool that cannot run says why when called.
func (s *Server) listTools() []map[string]any {
	out := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.tools {
		ann := map[string]any{"openWorldHint": true, "readOnlyHint": !t.Destructive}
		if t.Destructive {
			ann["destructiveHint"] = true
		}
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
			"annotations": ann,
		})
	}
	return out
}

func toolResult(text string, isErr bool) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isErr,
	}
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
		return nil, &rpcError{-32602, "invalid params"}
	}
	var tool *Tool
	for i := range s.tools {
		if s.tools[i].Name == p.Name {
			tool = &s.tools[i]
		}
	}
	if tool == nil {
		return nil, &rpcError{-32602, "unknown tool: " + p.Name}
	}
	if len(p.Arguments) == 0 {
		p.Arguments = json.RawMessage("{}")
	}
	start := time.Now()
	outcome := "ok"
	var result map[string]any
	switch {
	case !s.toolEnabled(*tool):
		outcome = "tool_disabled"
		result = toolResult("This tool is switched off by the Obelisk admin (Exposure → MCP gateway).", true)
	default:
		if ready, msg := tool.Backend(); !ready {
			outcome = "backend_missing"
			result = toolResult(msg, true)
		} else if text, terr := tool.Call(ctx, p.Arguments); terr != nil {
			outcome = terr.Code
			result = toolResult(terr.Msg, true)
		} else {
			result = toolResult(text, false)
		}
	}
	s.mu.Lock()
	s.calls++
	s.lastCall = time.Now()
	if outcome != "ok" {
		s.lastError = p.Name + ": " + outcome
	}
	s.mu.Unlock()
	log.Printf("mcp_call tool=%s outcome=%s duration_ms=%d", p.Name, outcome, time.Since(start).Milliseconds())
	return result, nil
}
