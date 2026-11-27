package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/mcpgateway"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type mcpConfigResponse struct {
	Enabled         bool                      `json:"enabled"`
	TokenConfigured bool                      `json:"token_configured"`
	PublicPath      string                    `json:"public_path"`
	LocalBaseURL    string                    `json:"local_base_url"`
	Runtime         mcpgateway.Status         `json:"runtime"`
	SendLog         []mcpgateway.SendLogEntry `json:"send_log"`
}

type patchMCPConfigRequest struct {
	Enabled         *bool           `json:"enabled,omitempty"`
	RegenerateToken *bool           `json:"regenerate_token,omitempty"`
	Tools           map[string]bool `json:"tools,omitempty"`
}

func (s *Server) buildMCPConfig(cfg string, spec obelisk.Spec) mcpConfigResponse {
	rt := mcpgateway.Status{}
	var logEntries []mcpgateway.SendLogEntry
	if s.MCPGateway != nil {
		rt = s.MCPGateway.Status()
		logEntries = s.MCPGateway.SendLog(20)
	}
	return mcpConfigResponse{
		Enabled:         spec.MCPGateway.Enabled,
		TokenConfigured: mcpgateway.TokenConfigured(cfg),
		PublicPath:      "/mcp",
		LocalBaseURL:    gatewayLocalBaseURL(s.MCPGatewayListenAddr),
		Runtime:         rt,
		SendLog:         logEntries,
	}
}

func (s *Server) reloadMCPGateway() {
	if s.MCPGateway != nil {
		_ = s.MCPGateway.Reload()
	}
}

func (s *Server) handleGetMCPConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.buildMCPConfig(cfg, spec))
}

func (s *Server) handlePatchMCPConfig(w http.ResponseWriter, r *http.Request) {
	var req patchMCPConfigRequest
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
	if len(req.Tools) > 0 {
		known := map[string]bool{}
		if s.MCPGateway != nil {
			for _, n := range s.MCPGateway.ToolNames() {
				known[n] = true
			}
		}
		if spec.MCPGateway.Tools == nil {
			spec.MCPGateway.Tools = map[string]bool{}
		}
		for name, on := range req.Tools {
			if !known[name] {
				http.Error(w, "unknown tool: "+name, http.StatusBadRequest)
				return
			}
			spec.MCPGateway.Tools[name] = on
		}
	}
	if req.Enabled != nil {
		spec.MCPGateway.Enabled = *req.Enabled
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.RegenerateToken != nil && *req.RegenerateToken {
		tok, err := mcpgateway.RotateToken(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.reloadMCPGateway()
		// The token is shown once; the client reloads config afterwards.
		writeJSON(w, map[string]string{"token": tok})
		return
	}
	s.reloadMCPGateway()
	writeJSON(w, s.buildMCPConfig(cfg, spec))
}

// handleMCPTest runs initialize, tools/list and (when ready) a sample search through the real listener.
func (s *Server) handleMCPTest(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	creds, _, err := mcpgateway.LoadCredentials(cfg)
	if err != nil || len(creds.Tokens) == 0 {
		http.Error(w, "generate an MCP token first", http.StatusBadRequest)
		return
	}
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !spec.MCPGateway.Enabled {
		http.Error(w, "turn the MCP gateway on first", http.StatusBadRequest)
		return
	}
	base := gatewayLocalBaseURL(s.MCPGatewayListenAddr)
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	call := func(body string) (map[string]any, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+creds.Tokens[0].Value)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	list, err := call(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tools/list failed: " + err.Error()})
		return
	}
	n := 0
	if res, ok := list["result"].(map[string]any); ok {
		if tools, ok := res["tools"].([]any); ok {
			n = len(tools)
		}
	}
	out := map[string]any{"ok": true, "tool_count": n}
	if s.MCPGateway != nil {
		for _, t := range s.MCPGateway.Status().Tools {
			if t.Name == "web_search" && t.Enabled && t.Ready {
				res, err := call(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"web_search","arguments":{"query":"obelisk","limit":1}}}`)
				if err != nil {
					out["search"] = "failed: " + err.Error()
				} else if rr, ok := res["result"].(map[string]any); ok && rr["isError"] == true {
					out["search"] = "error"
				} else {
					out["search"] = "ok"
				}
			}
		}
	}
	writeJSON(w, out)
}
