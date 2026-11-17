package obelisk

import (
	"regexp"
	"strings"
)

const (
	// LegacyDefaultInstanceID is the migrated single-instance id from service flags.
	LegacyDefaultInstanceID = "default"
	instanceIDLen           = 12
)

var instanceIDRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}[a-z0-9]$`)

// AllocateInstanceID returns a new globally unique, DNS/k8s-safe instance id.
func AllocateInstanceID(spec *Spec) string {
	for attempt := 0; attempt < 64; attempt++ {
		id := randomInstanceID()
		if !instanceIDValid(id) {
			continue
		}
		if _, _, taken := FindInstance(spec, id); taken {
			continue
		}
		return id
	}
	return "i" + randomDevSecret(instanceIDLen-1)
}

func randomInstanceID() string {
	// Lowercase hex from randomDevSecret; prefix with letter for k8s name rules.
	return "i" + randomDevSecret(instanceIDLen-1)
}

func instanceIDValid(id string) bool {
	id = strings.TrimSpace(id)
	if id == LegacyDefaultInstanceID {
		return true
	}
	return instanceIDRe.MatchString(id)
}
