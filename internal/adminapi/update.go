package adminapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/release"
)

const updateCacheTTL = 30 * time.Minute

// updateStatus mirrors `obelisk update --check`: the installed build, the latest published
// build, and whether an update is available. Applying the update (`sudo obelisk update`) must
// run on the host, so applying goes through a one-shot Job (see update_apply.go); the command
// stays as a manual fallback.
type updateStatus struct {
	InstalledBuild  int    `json:"installed_build"`
	LatestBuild     int    `json:"latest_build,omitempty"`
	Source          string `json:"source,omitempty"` // "release" or "dev"
	UpdateAvailable bool   `json:"update_available"`
	// Checkable is false when this install cannot be compared (no install meta, or a
	// source-built dev install that a release must not silently replace).
	Checkable bool      `json:"checkable"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error,omitempty"`
	Command   string    `json:"command"`
	// Apply is the in-cluster update Job, when one has been started from the UI.
	Apply *updateApply `json:"apply,omitempty"`
}

type updateCache struct {
	mu     sync.Mutex
	status updateStatus
	have   bool
}

// resolveLatest is replaced in tests.
var resolveLatest = func(repo string) (int, error) {
	return release.Resolve(release.BaseURL(), repo, "")
}

func (s *Server) checkUpdate(force bool) updateStatus {
	c := &s.updateCache
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.have && !force && time.Since(c.status.CheckedAt) < updateCacheTTL {
		return c.status
	}
	st := updateStatus{CheckedAt: time.Now().UTC(), Command: "sudo obelisk update"}
	meta, _, err := release.ReadInstallMeta(s.HelmRoot)
	switch {
	case errors.Is(err, os.ErrNotExist):
		st.Error = "install metadata not found; this appliance was not installed by obelisk install"
	case err != nil:
		st.Error = "cannot read install metadata: " + err.Error()
	default:
		st.InstalledBuild = meta.Build
		st.Source = meta.Source
		st.Checkable = true
		latest, rerr := resolveLatest(release.Repo(meta.Repo))
		switch {
		case errors.Is(rerr, release.ErrNoRelease):
			// Nothing published yet: not an error, not an update.
		case rerr != nil:
			st.Error = "cannot determine the latest release: " + rerr.Error()
		default:
			st.LatestBuild = latest
			// A dev install is only replaced with --force, so don't nag about it.
			st.UpdateAvailable = latest > meta.Build && meta.Source != "dev"
			if !st.UpdateAvailable && latest > meta.Build {
				st.Checkable = false
			}
		}
	}
	// Keep the last good answer on a transient lookup failure instead of dropping the alert.
	if st.Error != "" && c.have && c.status.Checkable && c.status.Error == "" {
		prev := c.status
		prev.Error = st.Error
		prev.CheckedAt = st.CheckedAt
		st = prev
	}
	c.status, c.have = st, true
	return st
}

// handleUpdateStatus serves GET /api/v1/update. ?refresh=1 bypasses the cache.
func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	st := s.checkUpdate(r.URL.Query().Get("refresh") == "1")
	if apply := currentUpdateApply(r.Context()); apply != nil {
		// A finished update makes the cached "update available" answer stale.
		if apply.Phase == "succeeded" && st.InstalledBuild < apply.TargetBuild {
			st = s.checkUpdate(true)
		}
		st.Apply = apply
	}
	_ = json.NewEncoder(w).Encode(st)
}
