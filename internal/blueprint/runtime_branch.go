package blueprint

import (
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// ensureUpstreamBranch rewrites legacy branch "main" to machine-<id> in obelisk.yaml.
func (rt *Runtime) ensureUpstreamBranch() {
	if rt == nil || rt.Sync == nil {
		return
	}
	_ = rt.Sync.UpdateSpec(func(s *obelisk.Spec) {
		if strings.TrimSpace(s.Blueprint.GitHub.Repository) == "" {
			return
		}
		if strings.TrimSpace(s.Blueprint.GitHub.Branch) != "main" {
			return
		}
		if target := obelisk.EffectiveBlueprintBranch(*s); target != "" && target != "main" {
			s.Blueprint.GitHub.Branch = target
		}
	})
}