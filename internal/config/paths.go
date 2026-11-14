package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath picks the active obelisk.yaml location.
// Order: OBELISK_CONFIG, existing DefaultPath, repo dev/obelisk.yaml, ~/.config/obelisk/obelisk.yaml.
func ResolvePath() string {
	if p := strings.TrimSpace(os.Getenv("OBELISK_CONFIG")); p != "" {
		return filepath.Clean(p)
	}
	if _, err := os.Stat(DefaultPath); err == nil {
		return DefaultPath
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

// EnsureReadablePath redirects a missing production path to ResolvePath() for local dev.
func EnsureReadablePath(requested string) string {
	if requested == "" {
		return ResolvePath()
	}
	if _, err := os.Stat(requested); err == nil {
		return requested
	} else if os.IsNotExist(err) && requested == DefaultPath {
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

func autoInitAllowed(path string) bool {
	return path != DefaultPath
}