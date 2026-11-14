package config

import (
	"os"
	"path/filepath"
)

// CredentialsDir is where gateway sidecar JSON files are stored (writable on appliance).
func CredentialsDir(configPath string) string {
	if d := os.Getenv("OBELISK_CREDENTIALS_DIR"); d != "" {
		return d
	}
	return filepath.Join(filepath.Dir(configPath), "credentials")
}