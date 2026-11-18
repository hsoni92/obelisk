package obelisk

import (
	"regexp"
	"strings"
)

const (
	// LegacyDefaultInstanceID is the migrated single-instance id from service flags.
	LegacyDefaultInstanceID = "default"
)

var instanceIDRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}[a-z0-9]$`)

// AllocateInstanceID returns a new globally unique, DNS/k8s-safe instance id.
func AllocateInstanceID(spec *Spec) string {
	return AllocateInstanceIDForType(spec, "")
}

// AllocateInstanceIDForType returns a unique id for a new instance of typ.
// When typ is empty, ids must be unique across all instances (legacy callers).
func AllocateInstanceIDForType(spec *Spec, typ string) string {
	for attempt := 0; attempt < 64; attempt++ {
		id := randomInstanceID()
		if !instanceIDValid(id) {
			continue
		}
		if typ != "" {
			if _, _, taken := FindInstanceByTypeAndID(spec, typ, id); taken {
				continue
			}
		} else if _, _, taken := FindInstance(spec, id); taken {
			continue
		}
		return id
	}
	return randomRouteSlug() + "-" + randomDevSecret(4)
}

func randomInstanceID() string {
	return randomRouteSlug()
}

func instanceIDValid(id string) bool {
	id = strings.TrimSpace(id)
	if id == LegacyDefaultInstanceID {
		return true
	}
	return instanceIDRe.MatchString(id)
}
