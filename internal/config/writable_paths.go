package config

import (
	"os"
	"path/filepath"
	"strings"
)

// WritableApplianceConfigPath is the live obelisk.yaml on k3s (hostPath under /var/lib/obelisk).
const WritableApplianceConfigPath = "/var/lib/obelisk/obelisk.yaml"

func applianceConfig(configPath string) bool {
	configPath = filepath.Clean(configPath)
	return strings.HasPrefix(configPath, "/etc/obelisk"+string(filepath.Separator)) ||
		configPath == "/etc/obelisk/obelisk.yaml"
}

// CredentialsDir is where gateway sidecar JSON files are stored (writable on appliance).
func CredentialsDir(configPath string) string {
	if d := os.Getenv("OBELISK_CREDENTIALS_DIR"); d != "" {
		return d
	}
	if applianceConfig(configPath) {
		if st, err := os.Stat("/var/lib/obelisk/credentials"); err == nil && st.IsDir() {
			return "/var/lib/obelisk/credentials"
		}
	}
	return filepath.Join(filepath.Dir(configPath), "credentials")
}

// LegacyCredentialsDir was used before k3s split config (read-only) from credentials.
func LegacyCredentialsDir(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "credentials")
}

// WriteConfigPath is where Save writes; on cluster appliances /etc/obelisk is read-only in the pod.
func WriteConfigPath(readPath string) string {
	if p := strings.TrimSpace(os.Getenv("OBELISK_CONFIG_WRITABLE")); p != "" {
		return filepath.Clean(p)
	}
	readPath = filepath.Clean(readPath)
	if applianceConfig(readPath) {
		return WritableApplianceConfigPath
	}
	return readPath
}

// ActiveConfigPath is the file Load should read (writable copy wins over /etc seed).
func ActiveConfigPath(requested string) string {
	if requested == "" {
		requested = ResolvePath()
	}
	requested = filepath.Clean(requested)
	wp := WriteConfigPath(requested)
	if wp != requested {
		if _, err := os.Stat(wp); err == nil {
			return wp
		}
	}
	return requested
}