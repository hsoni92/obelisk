package mail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hsoni92/obelisk/internal/config"
)

const credentialsFile = "mail.credentials.json"

// CredentialsFile stores gateway token and mailbox app passwords.
type CredentialsFile struct {
	GatewayToken string            `json:"gateway_token,omitempty"`
	Accounts     map[string]string `json:"accounts,omitempty"`
}

func credentialsPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), credentialsFile)
}

func LoadCredentials(configPath string) (CredentialsFile, bool, error) {
	p := credentialsPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return CredentialsFile{Accounts: map[string]string{}}, false, nil
		}
		return CredentialsFile{}, false, err
	}
	var c CredentialsFile
	if err := json.Unmarshal(data, &c); err != nil {
		return CredentialsFile{}, false, fmt.Errorf("parse mail credentials: %w", err)
	}
	if c.Accounts == nil {
		c.Accounts = map[string]string{}
	}
	ok := c.GatewayToken != "" || len(c.Accounts) > 0
	return c, ok, nil
}

func SaveCredentials(configPath string, c CredentialsFile) error {
	if c.Accounts == nil {
		c.Accounts = map[string]string{}
	}
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	p := credentialsPath(configPath)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func GatewayConfigured(configPath string) bool {
	c, ok, err := LoadCredentials(configPath)
	if err != nil || !ok {
		return false
	}
	return c.GatewayToken != ""
}