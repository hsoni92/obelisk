// Package syshealth checks the services and APIs the appliance depends on
// (GitHub, the network path to it, Tailscale, backup storage, OpenRouter) and
// reports what is wrong and what to do about it. The admin API and
// `obelisk status` both use it so they never disagree.
package syshealth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/tailscale"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// State is the outcome of one check.
type State string

const (
	StateOK   State = "ok"
	StateWarn State = "warn"
	StateFail State = "fail"
	// StateOff means the check does not apply (feature disabled or skipped).
	StateOff State = "off"
)

// Check groups.
const (
	GroupGitHub   = "github"
	GroupPlatform = "platform"
)

// Check is one row of the report.
type Check struct {
	ID        string `json:"id"`
	Group     string `json:"group"`
	Name      string `json:"name"`
	State     State  `json:"state"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail,omitempty"`
	Hint      string `json:"hint,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	// Blocking is set when this problem stops GitHub sign-in or onboarding.
	Blocking bool `json:"blocking,omitempty"`
	// GitHub carries the full API health for the github_api check.
	GitHub *blueprint.GitHubHealth `json:"github,omitempty"`
}

// Report is the full result.
type Report struct {
	CheckedAt string  `json:"checked_at"`
	Overall   State   `json:"overall"`
	Checks    []Check `json:"checks"`
}

// Options controls a Collect run.
type Options struct {
	// ConfigPath is the live obelisk.yaml the checks describe.
	ConfigPath string
	// Spec is the already-loaded config.
	Spec obelisk.Spec
	// Tokens reads the stored GitHub sign-in; defaults to a store at ConfigPath.
	Tokens *blueprint.TokenStore
	// OAuth is the OAuth app configuration.
	OAuth blueprint.OAuthConfig
	// GitHubAPI returns GitHub API health. The controller passes its cached
	// probe; the CLI leaves it nil and probes directly.
	GitHubAPI func(ctx context.Context) blueprint.GitHubHealth
	// GitHubOnly limits the run to the github group (public, pre-login view).
	GitHubOnly bool
	// Offline skips every network probe.
	Offline bool
	// LocalView marks a run from a shell (the CLI) rather than inside the
	// controller. The OAuth app may live only in the controller's environment,
	// so "not configured" is reported as a warning, not a failure.
	LocalView bool
	// Timeout bounds each check (default 6s).
	Timeout time.Duration
}

type checkFn func(ctx context.Context) Check

// Collect runs the checks concurrently and returns them in a stable order.
func Collect(ctx context.Context, opt Options) Report {
	if opt.Timeout <= 0 {
		opt.Timeout = 6 * time.Second
	}
	if opt.Tokens == nil {
		opt.Tokens = blueprint.NewTokenStore(opt.ConfigPath)
	}
	fns := []checkFn{
		opt.githubAPI,
		opt.githubOAuth,
		opt.githubBlueprint,
		opt.network,
	}
	if !opt.GitHubOnly {
		fns = append(fns, opt.configCheck, opt.tailscale, opt.backupRemote, opt.openRouter)
	}
	out := make([]Check, len(fns))
	var wg sync.WaitGroup
	for i, fn := range fns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, opt.Timeout)
			defer cancel()
			out[i] = fn(cctx)
		}()
	}
	wg.Wait()
	sort.SliceStable(out, func(i, j int) bool { return order(out[i]) < order(out[j]) })
	return Report{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Overall:   overall(out),
		Checks:    out,
	}
}

var idOrder = map[string]int{
	"github_api": 0, "github_oauth": 1, "github_blueprint": 2, "network": 3,
	"config": 4, "tailscale": 5, "backup_remote": 6, "openrouter": 7,
}

func order(c Check) int {
	if n, ok := idOrder[c.ID]; ok {
		return n
	}
	return len(idOrder)
}

func overall(checks []Check) State {
	worst := StateOK
	for _, c := range checks {
		switch c.State {
		case StateFail:
			return StateFail
		case StateWarn:
			worst = StateWarn
		}
	}
	return worst
}

// ---- GitHub -----------------------------------------------------------

func (o Options) githubAPI(ctx context.Context) Check {
	c := Check{ID: "github_api", Group: GroupGitHub, Name: "GitHub API"}
	if o.Offline {
		c.State, c.Summary = StateOff, "skipped (offline)"
		return c
	}
	var h blueprint.GitHubHealth
	if o.GitHubAPI != nil {
		h = o.GitHubAPI(ctx)
	} else {
		g := &blueprint.GitHub{Token: func(context.Context) (string, error) {
			tok, err := o.Tokens.Load()
			if err != nil || tok.AccessToken == "" {
				return "", fmt.Errorf("no GitHub token stored")
			}
			return tok.AccessToken, nil
		}}
		h = g.CheckAPI(ctx)
	}
	c.GitHub = &h
	c.LatencyMS = h.LatencyMS
	c.Blocking = h.Blocking
	c.Hint = h.Hint
	if h.State == blueprint.GHStateOK {
		c.State = StateOK
		c.Summary = "reachable"
		if h.Remaining != nil && h.Limit > 0 {
			c.Summary = fmt.Sprintf("reachable · quota %d/%d", *h.Remaining, h.Limit)
			if *h.Remaining*10 < h.Limit {
				c.State = StateWarn
				c.Hint = "Less than 10% of the hourly GitHub API quota is left."
			}
		}
		if !h.TokenPresent {
			c.Detail = "checked anonymously"
			if _, err := o.Tokens.Load(); errors.Is(err, os.ErrPermission) {
				// An anonymous check says nothing about the signed-in account's quota.
				c.State = StateWarn
				c.Detail = "checked anonymously: GitHub credentials are unreadable by this user"
				c.Hint = "Run with sudo to check the signed-in account's quota."
			}
		}
		return c
	}
	c.State = StateFail
	c.Summary = h.Title
	c.Detail = h.Message
	if h.HTTPStatus > 0 {
		c.Summary = fmt.Sprintf("%s (HTTP %d)", h.Title, h.HTTPStatus)
	}
	if h.ResetAt != "" {
		c.Detail = strings.TrimSpace(c.Detail + " Resets at " + h.ResetAt + ".")
	}
	if !h.Blocking {
		c.State = StateWarn
	}
	return c
}

func (o Options) githubOAuth(context.Context) Check {
	c := Check{ID: "github_oauth", Group: GroupGitHub, Name: "GitHub sign-in"}
	configured := o.OAuth.Configured()
	tok, err := o.Tokens.Load()
	hasToken := err == nil && tok.AccessToken != ""
	switch {
	case errors.Is(err, os.ErrPermission):
		c.State = StateWarn
		c.Summary = "credentials unreadable by this user"
		c.Hint = "Run with sudo to read the GitHub sign-in state."
	case !configured && !hasToken && o.LocalView:
		c.State = StateWarn
		c.Summary = "OAuth app not visible from this shell"
		c.Hint = "The controller may have it in its environment. If the console cannot sign in, set OBELISK_GITHUB_CLIENT_ID and OBELISK_GITHUB_CLIENT_SECRET on the controller (or Secret obelisk-github-oauth)."
	case !configured && !hasToken:
		c.State = StateFail
		c.Blocking = true
		c.Summary = "OAuth app not configured"
		c.Hint = "Set OBELISK_GITHUB_CLIENT_ID and OBELISK_GITHUB_CLIENT_SECRET on the controller (or Secret obelisk-github-oauth)."
	case !configured:
		c.State = StateWarn
		c.Summary = "OAuth app not configured; stored token cannot be refreshed"
		c.Hint = "Set OBELISK_GITHUB_CLIENT_ID and OBELISK_GITHUB_CLIENT_SECRET so the token can refresh."
	case !hasToken:
		c.State = StateWarn
		c.Summary = "OAuth app configured · not signed in"
		c.Hint = "Open the admin console and sign in with GitHub."
	default:
		c.State = StateOK
		c.Summary = "signed in"
		if tok.Login != "" {
			c.Summary = "signed in as " + tok.Login
		}
		var notes []string
		if tok.Scope != "" {
			notes = append(notes, "scopes "+tok.Scope)
			if !strings.Contains(tok.Scope, "repo") {
				c.State = StateWarn
				c.Hint = "The token lacks the repo scope; sign out and sign in again."
			}
		}
		if !tok.Expiry.IsZero() {
			left := time.Until(tok.Expiry)
			switch {
			case left <= 0 && tok.RefreshToken == "":
				c.State = StateFail
				c.Blocking = true
				c.Summary = "sign-in expired"
				c.Hint = "Sign in with GitHub again."
			case left <= 0:
				notes = append(notes, "access token expired; refreshes on next use")
			default:
				notes = append(notes, "access token valid for "+left.Round(time.Minute).String())
			}
		}
		c.Detail = strings.Join(notes, " · ")
	}
	return c
}

func (o Options) githubBlueprint(context.Context) Check {
	c := Check{ID: "github_blueprint", Group: GroupGitHub, Name: "Blueprint sync"}
	gh := o.Spec.Blueprint.GitHub
	tok, _ := o.Tokens.Load()
	if strings.TrimSpace(gh.Repository) == "" {
		if tok.AccessToken != "" {
			c.State = StateWarn
			c.Summary = "signed in but no blueprint repository in the live config"
			c.Hint = "The live config may have been reset (for example during an update). Restore it from the GitHub blueprint instead of onboarding as a new appliance."
		} else {
			c.State = StateOff
			c.Summary = "not set up"
		}
		return c
	}
	branch := obelisk.EffectiveBlueprintBranch(o.Spec)
	c.Summary = fmt.Sprintf("%s · %s@%s · gen %d", orDash(string(gh.SyncStatus)), gh.Repository, branch, gh.Generation)
	c.State = StateOK
	switch gh.SyncStatus {
	case obelisk.SyncFailed:
		c.State = StateFail
	case obelisk.SyncPushPaused, obelisk.SyncLocked:
		c.State = StateWarn
		c.Hint = "Unlock the blueprint in the admin console to resume pushes."
	}
	if gh.LastPushError != "" {
		c.Detail = gh.LastPushError
		if c.State == StateOK {
			c.State = StateWarn
		}
	}
	return c
}

func (o Options) network(ctx context.Context) Check {
	c := Check{ID: "network", Group: GroupGitHub, Name: "Network to GitHub"}
	if o.Offline {
		c.State, c.Summary = StateOff, "skipped (offline)"
		return c
	}
	start := time.Now()
	var failed []string
	for _, host := range []string{"api.github.com", "github.com"} {
		if err := dialTLS(ctx, host); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", host, shortErr(err)))
		}
	}
	c.LatencyMS = time.Since(start).Milliseconds()
	if len(failed) > 0 {
		c.State = StateFail
		c.Blocking = true
		c.Summary = "cannot reach GitHub over HTTPS"
		c.Detail = strings.Join(failed, "; ")
		c.Hint = "Check DNS, the outbound firewall or proxy for port 443, and the system clock."
		return c
	}
	c.State = StateOK
	c.Summary = "DNS and HTTPS to api.github.com and github.com"
	return c
}

// ---- Platform ---------------------------------------------------------

func (o Options) configCheck(context.Context) Check {
	c := Check{ID: "config", Group: GroupPlatform, Name: "Live config"}
	path := config.ActiveConfigPath(o.ConfigPath)
	st, err := os.Stat(path)
	if err != nil {
		c.State = StateFail
		c.Summary = "live config missing"
		c.Detail = path
		c.Hint = "Restore it from the GitHub blueprint or a backup; do not delete appliance-id."
		return c
	}
	c.Summary = fmt.Sprintf("machine %s · %d service instance(s)", orDash(o.Spec.Machine.ID), len(o.Spec.Instances))
	c.State = StateOK
	c.Detail = path + " · modified " + st.ModTime().Format(time.RFC3339)
	return c
}

func (o Options) tailscale(ctx context.Context) Check {
	c := Check{ID: "tailscale", Group: GroupPlatform, Name: "Tailscale"}
	if !o.Spec.Deployment.Tailscale.Enabled {
		c.State, c.Summary = StateOff, "not enabled"
		return c
	}
	if o.Offline {
		c.State, c.Summary = StateOff, "skipped (offline)"
		return c
	}
	ts := tailscale.GetStatus(ctx)
	switch {
	case ts.Connected:
		c.State = StateOK
		c.Summary = "connected"
		if ts.DNSName != "" {
			c.Summary = "connected · " + strings.TrimSuffix(ts.DNSName, ".")
		}
	case !ts.Installed:
		c.State = StateFail
		c.Summary = "enabled but not installed"
		c.Detail = ts.Error
	default:
		c.State = StateWarn
		c.Summary = "enabled but not connected (" + orDash(ts.BackendState) + ")"
		c.Detail = ts.Error
		if ts.AuthURL != "" {
			c.Hint = "Authenticate this node: " + ts.AuthURL
		}
	}
	return c
}

func (o Options) backupRemote(ctx context.Context) Check {
	c := Check{ID: "backup_remote", Group: GroupPlatform, Name: "Backup storage"}
	var endpoints []string
	for _, d := range o.Spec.Backup.Destinations {
		if d.Enabled && strings.TrimSpace(d.Endpoint) != "" {
			endpoints = append(endpoints, d.Endpoint)
		}
	}
	if len(endpoints) == 0 && strings.TrimSpace(o.Spec.Backup.Remote.Endpoint) != "" {
		endpoints = append(endpoints, o.Spec.Backup.Remote.Endpoint)
	}
	if len(endpoints) == 0 {
		c.State, c.Summary = StateOff, "no remote destination configured"
		return c
	}
	if o.Offline {
		c.State, c.Summary = StateOff, "skipped (offline)"
		return c
	}
	var bad []string
	for _, ep := range endpoints {
		if err := dialEndpoint(ctx, ep); err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", ep, shortErr(err)))
		}
	}
	if len(bad) > 0 {
		c.State = StateFail
		c.Summary = fmt.Sprintf("%d of %d destination(s) unreachable", len(bad), len(endpoints))
		c.Detail = strings.Join(bad, "; ")
		c.Hint = "Backups to unreachable destinations will fail. Check the endpoint and network path."
		return c
	}
	c.State = StateOK
	c.Summary = fmt.Sprintf("%d destination(s) reachable", len(endpoints))
	return c
}

func (o Options) openRouter(ctx context.Context) Check {
	c := Check{ID: "openrouter", Group: GroupPlatform, Name: "OpenRouter"}
	if !o.Spec.OpenRouterGateway.Enabled {
		c.State, c.Summary = StateOff, "gateway not enabled"
		return c
	}
	if o.Offline {
		c.State, c.Summary = StateOff, "skipped (offline)"
		return c
	}
	start := time.Now()
	if err := dialTLS(ctx, "openrouter.ai"); err != nil {
		c.State = StateFail
		c.Summary = "cannot reach openrouter.ai"
		c.Detail = shortErr(err)
		c.Hint = "The OpenRouter gateway cannot forward requests until this is fixed."
		return c
	}
	c.LatencyMS = time.Since(start).Milliseconds()
	c.State = StateOK
	c.Summary = "reachable"
	return c
}

// ---- helpers ----------------------------------------------------------

func dialTLS(ctx context.Context, host string) error {
	d := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, "443"))
	if err != nil {
		return err
	}
	return conn.Close()
}

func dialEndpoint(ctx context.Context, endpoint string) error {
	raw := strings.TrimSpace(endpoint)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("invalid endpoint")
	}
	port := u.Port()
	if port == "" {
		port = "443"
		if u.Scheme == "http" {
			port = "80"
		}
	}
	d := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(u.Hostname(), port))
	if err != nil {
		return err
	}
	return conn.Close()
}

// shortErr trims dial errors to the cause (for example "connection refused").
func shortErr(err error) string {
	var oe *net.OpError
	if errors.As(err, &oe) && oe.Err != nil {
		return oe.Err.Error()
	}
	return err.Error()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
