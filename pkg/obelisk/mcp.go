package obelisk

import "strings"

// MCPGateway configures the built-in MCP server (tools for AI editors and agents).
// The bearer token lives in mcp.credentials.json, not in obelisk.yaml.
type MCPGateway struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Tools holds per-tool on/off switches by tool name. A missing name uses the tool's default.
	// Unknown names are kept so a newer config survives an older binary.
	Tools map[string]bool `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// NormalizeMCP trims tool names and drops empty keys.
func NormalizeMCP(spec *Spec) {
	if len(spec.MCPGateway.Tools) == 0 {
		spec.MCPGateway.Tools = nil
		return
	}
	out := make(map[string]bool, len(spec.MCPGateway.Tools))
	for k, v := range spec.MCPGateway.Tools {
		k = strings.TrimSpace(k)
		if k != "" {
			out[k] = v
		}
	}
	spec.MCPGateway.Tools = out
}
