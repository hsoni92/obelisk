package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath picks the active obelisk.yaml location.
// Order: OBELISK_CONFIG, live appliance file (/var/lib), /etc seed, dev/obelisk.yaml, ~/.config.
func ResolvePath() string {
	if p := strings.TrimSpace(os.Getenv("OBELISK_CONFIG")); p != "" {
		return filepath.Clean(p)
	}
	if live := WriteConfigPath(DefaultPath); live != DefaultPath {
		if _, err := os.Stat(live); err == nil {
			return live
		}
	}
	if _, err := os.Stat(DefaultPath); err == nil {
		return ActiveConfigPath(DefaultPath)
	}
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, "deploy", "helm", "platform")); err == nil {
			return filepath.Join(wd, "dev", "obelisk.yaml")
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".config", "obelisk", "obelisk.yaml")
	}
	return filepath.Join(os.TempDir(), "obelisk", "obelisk.yaml")
}

// EnsureReadablePath picks the file the API and controller should read.
// On appliances, ActiveConfigPath wins (/var/lib copy over /etc seed). When /etc
// is missing entirely (common on k3s hostPath), fall back to ResolvePath() for dev.
func EnsureReadablePath(requested string) string {
	if requested == "" {
		requested = ResolvePath()
	}
	requested = filepath.Clean(requested)
	if applianceConfig(requested) {
		live := ActiveConfigPath(requested)
		if live != requested {
			return live
		}
		if _, err := os.Stat(requested); err == nil {
			return requested
		}
		if _, err := os.Stat(WritableApplianceConfigPath); err == nil {
			return WritableApplianceConfigPath
		}
	}
	_, err := os.Stat(requested)
	if err == nil {
		return requested
	}
	if os.IsNotExist(err) && requested == DefaultPath {
		return ResolvePath()
	}
	return requested
}

// ResolveHelmRoot is the checkout path containing deploy/helm (dev: repo root).
func ResolveHelmRoot() string {
	if r := strings.TrimSpace(os.Getenv("OBELISK_ROOT")); r != "" {
		return r
	}
	if _, err := os.Stat("deploy/helm/platform"); err == nil {
		wd, err := os.Getwd()
		if err == nil {
			return wd
		}
	}
	return "/opt/obelisk"
}

func defaultProfileForPath(path string) string {
	if path == DefaultPath {
		return "vps"
	}
	if p := os.Getenv("OBELISK_PROFILE"); p == "local" || p == "vps" {
		return p
	}
	return "local"
}

// existingApplianceState reports whether path is the live appliance config and sits next to
// state that only a provisioned appliance has (appliance-id or credential files).
func existingApplianceState(path string) bool {
	if filepath.Clean(path) != WritableApplianceConfigPath {
		return false
	}
	return provisionedStateDir(filepath.Dir(path))
}

func provisionedStateDir(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "appliance-id")); err == nil {
		return true
	}
	entries, err := os.ReadDir(filepath.Join(dir, "credentials"))
	return err == nil && len(entries) > 0
}

func autoInitAllowed(path string) bool {
	return path != DefaultPath
}
