package backup

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hsoni92/obelisk/internal/config"
)

// RemoteCredentials are never written to obelisk.yaml (host-restricted file or K8s Secret in cluster).
type RemoteCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func credentialsPath(configPath string) string {
	return config.CredentialsDir(configPath) + "/backup-remote.credentials.json"
}

// LoadCredentials reads backup remote API keys from disk beside obelisk.yaml.
func LoadCredentials(configPath string) (RemoteCredentials, bool, error) {
	p := credentialsPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return RemoteCredentials{}, false, nil
		}
		return RemoteCredentials{}, false, err
	}
	var c RemoteCredentials
	if err := json.Unmarshal(data, &c); err != nil {
		return RemoteCredentials{}, false, fmt.Errorf("parse backup credentials: %w", err)
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return RemoteCredentials{}, false, nil
	}
	return c, true, nil
}

// SaveCredentials persists keys with mode 0600 next to the config file.
func SaveCredentials(configPath string, c RemoteCredentials) error {
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return fmt.Errorf("access_key_id and secret_access_key are required")
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