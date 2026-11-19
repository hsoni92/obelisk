package discovery

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
)

const credentialsFile = "discovery.projects.json"

type credentialsOnDisk struct {
	Tokens map[string]string `json:"tokens,omitempty"`
}

func credentialsPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), credentialsFile)
}

func loadCredentials(configPath string) (credentialsOnDisk, error) {
	p := credentialsPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return credentialsOnDisk{Tokens: map[string]string{}}, nil
		}
		return credentialsOnDisk{}, fmt.Errorf("read discovery credentials: %w", err)
	}
	var c credentialsOnDisk
	if err := json.Unmarshal(data, &c); err != nil {
		return credentialsOnDisk{}, fmt.Errorf("parse discovery credentials: %w", err)
	}
	if c.Tokens == nil {
		c.Tokens = map[string]string{}
	}
	return c, nil
}

func saveCredentials(configPath string, c credentialsOnDisk) error {
	if c.Tokens == nil {
		c.Tokens = map[string]string{}
	}
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	p := credentialsPath(configPath)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write discovery credentials: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		return fmt.Errorf("commit discovery credentials: %w", err)
	}
	return nil
}

// ReadToken returns the bearer token for a project, or "" if unset.
func ReadToken(configPath, projectID string) string {
	c, err := loadCredentials(configPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(c.Tokens[projectID])
}

// WriteToken stores the bearer token for a project.
func WriteToken(configPath, projectID, token string) error {
	c, err := loadCredentials(configPath)
	if err != nil {
		return err
	}
	c.Tokens[projectID] = token
	return saveCredentials(configPath, c)
}

// DeleteToken removes a project's token.
func DeleteToken(configPath, projectID string) {
	c, err := loadCredentials(configPath)
	if err != nil {
		return
	}
	delete(c.Tokens, projectID)
	_ = saveCredentials(configPath, c)
}

// ValidToken compares bearer tokens in constant time.
func ValidToken(stored, provided string) bool {
	stored = strings.TrimSpace(stored)
	provided = strings.TrimSpace(provided)
	if stored == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(provided)) == 1
}