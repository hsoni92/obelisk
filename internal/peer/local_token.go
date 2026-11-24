package peer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
)

const localCredFile = "peer.local.json"

type localCreds struct {
	Token string `json:"token,omitempty"`
}

func localCredPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), localCredFile)
}

// ReadLocalToken returns the peer bearer token for advertise calls.
func ReadLocalToken(configPath string) string {
	p := localCredPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	var c localCreds
	if json.Unmarshal(data, &c) != nil {
		return ""
	}
	return strings.TrimSpace(c.Token)
}

// ClearLocalToken removes a stale peer token so the next advertise may re-admit.
func ClearLocalToken(configPath string) {
	_ = os.Remove(localCredPath(configPath))
}

// WriteLocalToken stores the peer bearer token from the gateway.
func WriteLocalToken(configPath, token string) error {
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("empty token")
	}
	data, err := json.Marshal(localCreds{Token: token})
	if err != nil {
		return err
	}
	p := localCredPath(configPath)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
