package obelisk

import (
	"fmt"
	"strings"
)

const OpenRouterUpstreamBase = "https://openrouter.ai"

// OpenRouterGateway configures multi-account round-robin proxying to OpenRouter.
type OpenRouterGateway struct {
	Enabled  bool                `yaml:"enabled" json:"enabled"`
	Accounts []OpenRouterAccount `yaml:"accounts,omitempty" json:"accounts,omitempty"`
}

// OpenRouterAccount is metadata in obelisk.yaml; API keys live in openrouter.credentials.json.
type OpenRouterAccount struct {
	ID             string `yaml:"id" json:"id"`
	Label          string `yaml:"label" json:"label"`
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Notes          string `yaml:"notes,omitempty" json:"notes,omitempty"`
	KeyFingerprint string `yaml:"key_fingerprint,omitempty" json:"key_fingerprint,omitempty"`
	UpstreamLabel  string `yaml:"upstream_label,omitempty" json:"upstream_label,omitempty"`
	KeyTail        string `yaml:"key_tail,omitempty" json:"key_tail,omitempty"`
	UpstreamUserID string `yaml:"upstream_user_id,omitempty" json:"upstream_user_id,omitempty"`
}

// FindAccountByFingerprint returns an existing account with the same upstream key fingerprint.
func FindAccountByFingerprint(accounts []OpenRouterAccount, fingerprint string, excludeID string) (OpenRouterAccount, bool) {
	if fingerprint == "" {
		return OpenRouterAccount{}, false
	}
	for _, a := range accounts {
		if excludeID != "" && a.ID == excludeID {
			continue
		}
		if a.KeyFingerprint == fingerprint {
			return a, true
		}
	}
	return OpenRouterAccount{}, false
}

// NormalizeOpenRouter ensures account ids and trims labels.
func NormalizeOpenRouter(spec *Spec) {
	spec.OpenRouterGateway.Accounts = trimOpenRouterAccounts(spec.OpenRouterGateway.Accounts)
}

func trimOpenRouterAccounts(list []OpenRouterAccount) []OpenRouterAccount {
	out := make([]OpenRouterAccount, 0, len(list))
	seen := make(map[string]struct{})
	for _, a := range list {
		id := strings.TrimSpace(a.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		a.ID = id
		a.Label = strings.TrimSpace(a.Label)
		if a.Label == "" {
			a.Label = id
		}
		out = append(out, a)
	}
	return out
}

// ValidateOpenRouter returns an error when enabled but misconfigured.
func ValidateOpenRouter(g OpenRouterGateway) error {
	if !g.Enabled {
		return nil
	}
	enabled := 0
	for _, a := range g.Accounts {
		if a.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return fmt.Errorf("openrouter_gateway: enabled but no enabled accounts")
	}
	return nil
}
