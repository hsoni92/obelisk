package adminapi

import (
	"context"
	"net/http"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/syshealth"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// handleSystemHealth reports GitHub and supporting-service health. It is public
// so the sign-in and onboarding screens can explain a GitHub failure before
// anyone is signed in; unauthenticated callers only get the github group.
// ?fresh=1 forces a new GitHub probe (throttled; /rate_limit is not metered).
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	authed := false
	if s.Blueprint != nil {
		_, authed = s.Blueprint.Authenticate(r)
	}

	cfg := config.EnsureReadablePath(s.ConfigPath)
	var spec obelisk.Spec
	if loaded, err := config.Load(cfg); err == nil {
		spec = loaded
	}

	opt := syshealth.Options{
		ConfigPath: cfg,
		Spec:       spec,
		GitHubOnly: !authed,
	}
	if s.Blueprint != nil {
		fresh := r.URL.Query().Get("fresh") == "1"
		opt.Tokens = s.Blueprint.Tokens
		opt.OAuth = s.Blueprint.OAuth.Config
		opt.GitHubAPI = func(ctx context.Context) blueprint.GitHubHealth {
			return s.Blueprint.GitHubAPIHealth(ctx, fresh)
		}
	}
	report := syshealth.Collect(r.Context(), opt)
	if !authed {
		report = redactPublicReport(report)
	}
	writeJSON(w, report)
}

// redactPublicReport keeps states and hints but drops identities (the stored
// GitHub login, repository and branch names, last push errors) that must not
// be visible before sign-in.
func redactPublicReport(r syshealth.Report) syshealth.Report {
	out := r
	out.Checks = make([]syshealth.Check, len(r.Checks))
	copy(out.Checks, r.Checks)
	for i := range out.Checks {
		c := &out.Checks[i]
		switch c.ID {
		case "github_oauth":
			c.Detail = ""
			if c.State == syshealth.StateOK {
				c.Summary = "signed in"
			}
		case "github_blueprint":
			c.Detail = ""
			if c.State != syshealth.StateOff && c.Summary != "" {
				c.Summary = "configured"
				if c.State == syshealth.StateFail {
					c.Summary = "sync failing"
				}
			}
		}
	}
	return out
}
