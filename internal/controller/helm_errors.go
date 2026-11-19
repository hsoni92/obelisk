package controller

import (
	"strings"
)

// HelmReleaseNotFound reports whether helm status/history output indicates a missing release.
func HelmReleaseNotFound(combinedOutput []byte, err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(string(combinedOutput))
	if strings.Contains(text, "not found") {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}