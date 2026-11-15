package blueprint

import (
	"fmt"
	"regexp"
)

var (
	pemRE    = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]+-----`)
	secretRE = regexp.MustCompile(`(?i)(password|secret_key|secret_access_key|api_key|access_key_id)\s*:\s*(.+)$`)
	tokenRE  = regexp.MustCompile(`ghp_[A-Za-z0-9]+|github_pat_[A-Za-z0-9_]+|AKIA[0-9A-Z]{16}`)
)

// ScanPublicYAML rejects secret-shaped material in files that would be committed in cleartext.
func ScanPublicYAML(b []byte) error {
	if pemRE.Match(b) {
		return fmt.Errorf("pre-push scan: PEM block in public blueprint")
	}
	if tokenRE.Match(b) {
		return fmt.Errorf("pre-push scan: token-like value in public blueprint")
	}
	for _, line := range splitLines(b) {
		m := secretRE.FindSubmatch(line)
		if m == nil {
			continue
		}
		val := string(m[2])
		if isEmptyYAML(val) {
			continue
		}
		return fmt.Errorf("pre-push scan: secret field %q in public blueprint", string(m[1]))
	}
	return nil
}

func splitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			lines = append(lines, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, b[start:])
	}
	return lines
}

func isEmptyYAML(val string) bool {
	switch val {
	case "", `""`, "''", "~", "null", "Null", "NULL", "{}", "[]":
		return true
	default:
		return false
	}
}
