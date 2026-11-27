package mcpgateway

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
)

const (
	credentialsFile = "mcp.credentials.json"
	sendLogFile     = "mcp.sendlog.json"
	// TokenPrefix marks MCP tokens so they are easy to spot in leak scanners.
	TokenPrefix = "obk_mcp_"
)

// Token is one bearer token. v1 uses one token; the file stores a list so
// per-client tokens can be added later without a format change.
type Token struct {
	ID        string `json:"id"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CredentialsFile is mcp.credentials.json.
type CredentialsFile struct {
	Tokens []Token `json:"tokens,omitempty"`
}

func credentialsPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), credentialsFile)
}

// LoadCredentials reads the sidecar; ok is true when at least one token exists.
func LoadCredentials(configPath string) (CredentialsFile, bool, error) {
	data, err := os.ReadFile(credentialsPath(configPath))
	if err != nil {
		if os.IsNotExist(err) {
			return CredentialsFile{}, false, nil
		}
		return CredentialsFile{}, false, err
	}
	var c CredentialsFile
	if err := json.Unmarshal(data, &c); err != nil {
		return CredentialsFile{}, false, fmt.Errorf("parse mcp credentials: %w", err)
	}
	return c, len(c.Tokens) > 0, nil
}

// SaveCredentials writes the sidecar atomically with mode 0600.
func SaveCredentials(configPath string, c CredentialsFile) error {
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

// TokenConfigured reports whether a token exists on disk.
func TokenConfigured(configPath string) bool {
	_, ok, err := LoadCredentials(configPath)
	return err == nil && ok
}

// NewToken returns a fresh random token string.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return TokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// RotateToken replaces all tokens with one new token and returns its value.
// Old clients stop working on the next request; there is no grace period.
func RotateToken(configPath string) (string, error) {
	tok, err := NewToken()
	if err != nil {
		return "", err
	}
	c := CredentialsFile{Tokens: []Token{{
		ID:        "default",
		Value:     tok,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}}}
	if err := SaveCredentials(configPath, c); err != nil {
		return "", err
	}
	return tok, nil
}
