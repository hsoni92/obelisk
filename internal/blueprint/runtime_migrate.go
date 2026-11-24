package blueprint

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func (rt *Runtime) maybeMigrateMainBranch(ctx context.Context) {
	if rt == nil || rt.Sync == nil {
		return
	}
	spec, err := config.Load(rt.ConfigPath)
	if err != nil || spec.Blueprint.GitHub.Repository == "" {
		return
	}
	target, err := obelisk.BlueprintUpstreamBranch(spec.Machine.ID)
	if err != nil {
		return
	}
	cur := strings.TrimSpace(spec.Blueprint.GitHub.Branch)
	if cur == target {
		return
	}
	if cur != "" && cur != "main" {
		return
	}
	if !rt.Session.Unlocked() {
		return
	}
	pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	files, ferr := rt.Git.Fetch(pctx, spec.Blueprint.GitHub.Repository, "main")
	if ferr != nil {
		return
	}
	var manifest Manifest
	if err := json.Unmarshal(files[FileManifest], &manifest); err != nil {
		return
	}
	if strings.TrimSpace(manifest.MachineID) != strings.TrimSpace(spec.Machine.ID) {
		return
	}
	_ = rt.Sync.UpdateSpec(func(s *obelisk.Spec) {
		s.Blueprint.GitHub.Branch = target
		s.Blueprint.GitHub.SyncStatus = obelisk.SyncPending
	})
	if err := rt.Sync.PushNow(pctx); err != nil {
		_ = rt.Sync.fail(err)
		return
	}
	rt.InvalidateProbe()
}
