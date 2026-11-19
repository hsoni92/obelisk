package obelisk

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const DiscoveryPathPrefix = "/discover/v1/projects/"

// NewProjectID returns an opaque stable id for URLs.
func NewProjectID() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return "prj_" + base64.RawURLEncoding.EncodeToString(b)
}

// NormalizeProjects fills missing ids, trims fields, drops legacy-only rows with no id/name.
func NormalizeProjects(projects []Project) []Project {
	out := make([]Project, 0, len(projects))
	seenID := map[string]bool{}
	seenName := map[string]bool{}
	for _, p := range projects {
		p.Name = strings.TrimSpace(p.Name)
		if p.ID == "" && p.Name != "" {
			p.ID = slugProjectName(p.Name)
		}
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" || p.Name == "" {
			continue
		}
		if seenID[p.ID] || seenName[strings.ToLower(p.Name)] {
			continue
		}
		seenID[p.ID] = true
		seenName[strings.ToLower(p.Name)] = true
		p.PostgresDB = ""
		p.RedisDB = 0
		ids := make([]string, 0, len(p.InstanceIDs))
		for _, ref := range p.InstanceIDs {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				ids = append(ids, ref)
			}
		}
		p.InstanceIDs = ids
		out = append(out, p)
	}
	return out
}

func slugProjectName(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return NewProjectID()
	}
	return s
}

// FindProject returns the project with id or nil.
func FindProject(spec Spec, id string) *Project {
	id = strings.TrimSpace(id)
	for i := range spec.Projects {
		if spec.Projects[i].ID == id {
			return &spec.Projects[i]
		}
	}
	return nil
}

// ValidateProjectInstanceRefs ensures every ref exists on spec.instances (enabled or not).
func ValidateProjectInstanceRefs(spec Spec, p Project) error {
	known := map[string]bool{}
	for _, inst := range spec.Instances {
		known[inst.StateKey()] = true
		known[inst.Type+"/"+inst.ID] = true
	}
	for _, ref := range p.InstanceIDs {
		if !known[ref] {
			return fmt.Errorf("unknown instance %q", ref)
		}
	}
	return nil
}

// ProjectInstanceSet returns a set of instance state keys for membership filtering.
func ProjectInstanceSet(p Project) map[string]bool {
	set := make(map[string]bool, len(p.InstanceIDs))
	for _, ref := range p.InstanceIDs {
		set[ref] = true
	}
	return set
}