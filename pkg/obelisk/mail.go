package obelisk

import (
	"fmt"
	"strings"
)

// MailGateway configures pooled SMTP send via HTTP gateway.
type MailGateway struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Accounts []MailAccount `yaml:"accounts,omitempty" json:"accounts,omitempty"`
}

// MailAccount metadata in obelisk.yaml; secrets in mail.credentials.json.
type MailAccount struct {
	ID       string `yaml:"id" json:"id"`
	Label    string `yaml:"label" json:"label"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Address  string `yaml:"address" json:"address"`
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Notes    string `yaml:"notes,omitempty" json:"notes,omitempty"`
}

func NormalizeMail(spec *Spec) {
	out := make([]MailAccount, 0, len(spec.MailGateway.Accounts))
	seen := make(map[string]struct{})
	for _, a := range spec.MailGateway.Accounts {
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
		a.Address = strings.ToLower(strings.TrimSpace(a.Address))
		if a.Provider == "" && a.Address != "" {
			a.Provider = MailboxDomain(a.Address)
		}
		out = append(out, a)
	}
	spec.MailGateway.Accounts = out
}

func MailboxDomain(email string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(email, "@")[1]))
}

// FindMailAccountByAddress returns account with same address (case-insensitive).
func FindMailAccountByAddress(accounts []MailAccount, address string, excludeID string) (MailAccount, bool) {
	addr := strings.ToLower(strings.TrimSpace(address))
	for _, a := range accounts {
		if excludeID != "" && a.ID == excludeID {
			continue
		}
		if strings.EqualFold(a.Address, addr) {
			return a, true
		}
	}
	return MailAccount{}, false
}

// ValidateMail returns an error when enabled but misconfigured.
func ValidateMail(g MailGateway) error {
	if !g.Enabled {
		return nil
	}
	n := 0
	for _, a := range g.Accounts {
		if a.Enabled {
			n++
		}
	}
	if n == 0 {
		return fmt.Errorf("mail_gateway: enabled but no enabled accounts")
	}
	return nil
}
