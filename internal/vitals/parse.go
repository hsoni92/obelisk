package vitals

import (
	"regexp"
	"strconv"
	"strings"
)

var quantRe = regexp.MustCompile(`^(\d+)([a-zA-Z]*)$`)

// parseBytesQuantity converts k8s style quantities (Ki, Mi, Gi) and plain integers to bytes.
func parseBytesQuantity(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if m := quantRe.FindStringSubmatch(s); m != nil {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return 0, false
		}
		switch strings.ToLower(m[2]) {
		case "", "b":
			return n, true
		case "ki":
			return n * 1024, true
		case "mi":
			return n * 1024 * 1024, true
		case "gi":
			return n * 1024 * 1024 * 1024, true
		case "ti":
			return n * 1024 * 1024 * 1024 * 1024, true
		case "k":
			return n * 1000, true
		case "m":
			return n * 1000 * 1000, true
		case "g":
			return n * 1000 * 1000 * 1000, true
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

// parseCPUQuantity returns millicores.
func parseCPUQuantity(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if strings.HasSuffix(s, "m") {
		n, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		return n, err == nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n * 1000, true
}
