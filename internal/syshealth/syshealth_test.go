package syshealth

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func intp(n int) *int { return &n }

func byID(t *testing.T, r Report, id string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("check %q missing from report: %+v", id, r.Checks)
	return Check{}
}

// newOpts builds offline options over a temp config with a token store.
func newOpts(t *testing.T, mutate func(*obelisk.Spec)) (Options, *blueprint.TokenStore) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-health01"
	if mutate != nil {
		mutate(&spec)
	}
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}
	tokens := blueprint.NewTokenStore(path)
	return Options{
		ConfigPath: path,
		Spec:       spec,
		Tokens:     tokens,
		OAuth:      blueprint.OAuthConfig{ClientID: "id", ClientSecret: "secret"},
		GitHubAPI: func(context.Context) blueprint.GitHubHealth {
			return blueprint.ClassifyGitHubError(nil, time.Now())
		},
		Offline: true,
	}, tokens
}

func TestGitHubAPICheck_rateLimited(t *testing.T) {
	opt, _ := newOpts(t, nil)
	opt.Offline = false
	opt.GitHubAPI = func(context.Context) blueprint.GitHubHealth {
		return blueprint.ClassifyGitHubError(&blueprint.StatusError{
			Code: 403, Body: `{"message":"API rate limit exceeded"}`,
			ResetAt: time.Now().Add(20 * time.Minute), Remaining: 0,
		}, time.Now())
	}
	c := opt.githubAPI(context.Background())
	if c.State != StateFail || !c.Blocking {
		t.Fatalf("check = %+v", c)
	}
	if c.GitHub == nil || c.GitHub.State != blueprint.GHStateRateLimited {
		t.Fatalf("github detail missing: %+v", c.GitHub)
	}
	if c.Summary == "" || c.Hint == "" {
		t.Fatalf("summary and hint required: %+v", c)
	}
}

func TestGitHubAPICheck_quota(t *testing.T) {
	opt, _ := newOpts(t, nil)
	opt.Offline = false
	for _, tt := range []struct {
		remaining int
		want      State
	}{{4000, StateOK}, {200, StateWarn}} {
		opt.GitHubAPI = func(context.Context) blueprint.GitHubHealth {
			h := blueprint.ClassifyGitHubError(nil, time.Now())
			h.Limit, h.Remaining, h.TokenPresent = 5000, intp(tt.remaining), true
			return h
		}
		if c := opt.githubAPI(context.Background()); c.State != tt.want {
			t.Fatalf("remaining %d: state = %s, want %s (%+v)", tt.remaining, c.State, tt.want, c)
		}
	}
}

func TestGitHubOAuthCheck(t *testing.T) {
	opt, tokens := newOpts(t, nil)
	ctx := context.Background()

	opt.OAuth = blueprint.OAuthConfig{}
	if c := opt.githubOAuth(ctx); c.State != StateFail || !c.Blocking {
		t.Fatalf("unconfigured app with no token = %+v", c)
	}

	opt.LocalView = true
	if c := opt.githubOAuth(ctx); c.State != StateWarn || c.Blocking {
		t.Fatalf("from a shell the OAuth app may live only in the controller env: %+v", c)
	}
	opt.LocalView = false

	opt.OAuth = blueprint.OAuthConfig{ClientID: "id"}
	if c := opt.githubOAuth(ctx); c.State != StateWarn {
		t.Fatalf("configured, not signed in = %+v", c)
	}

	if err := tokens.Save(blueprint.Token{AccessToken: "t", Login: "octo", Scope: "read:user,repo", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if c := opt.githubOAuth(ctx); c.State != StateOK || c.Summary != "signed in as octo" {
		t.Fatalf("valid token = %+v", c)
	}

	if err := tokens.Save(blueprint.Token{AccessToken: "t", Login: "octo", Scope: "read:user", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if c := opt.githubOAuth(ctx); c.State != StateWarn {
		t.Fatalf("missing repo scope should warn: %+v", c)
	}

	if err := tokens.Save(blueprint.Token{AccessToken: "t", Login: "octo", Scope: "repo", Expiry: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if c := opt.githubOAuth(ctx); c.State != StateFail || !c.Blocking {
		t.Fatalf("expired token without refresh = %+v", c)
	}
}

// The incident that motivated this package: after an update the live config
// lost its blueprint section while the GitHub sign-in survived.
func TestGitHubBlueprintCheck_configLostButSignedIn(t *testing.T) {
	opt, tokens := newOpts(t, nil)
	if err := tokens.Save(blueprint.Token{AccessToken: "t", Login: "octo"}); err != nil {
		t.Fatal(err)
	}
	c := opt.githubBlueprint(context.Background())
	if c.State != StateWarn {
		t.Fatalf("check = %+v, want warn", c)
	}
	if c.Hint == "" {
		t.Fatal("must tell the operator to restore instead of onboarding")
	}
}

func TestGitHubBlueprintCheck_states(t *testing.T) {
	ctx := context.Background()
	opt, _ := newOpts(t, nil)
	if c := opt.githubBlueprint(ctx); c.State != StateOff {
		t.Fatalf("never set up = %+v", c)
	}
	for _, tt := range []struct {
		sync string
		want State
	}{
		{obelisk.SyncSynced, StateOK},
		{obelisk.SyncFailed, StateFail},
		{obelisk.SyncPushPaused, StateWarn},
	} {
		opt, _ := newOpts(t, func(s *obelisk.Spec) {
			s.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
			s.Blueprint.GitHub.Branch = "main"
			s.Blueprint.GitHub.SyncStatus = tt.sync
		})
		if c := opt.githubBlueprint(ctx); c.State != tt.want {
			t.Fatalf("sync %s: state = %s, want %s", tt.sync, c.State, tt.want)
		}
	}
}

func TestConfigCheck(t *testing.T) {
	opt, _ := newOpts(t, nil)
	if c := opt.configCheck(context.Background()); c.State != StateOK {
		t.Fatalf("existing config = %+v", c)
	}
	if err := os.Remove(opt.ConfigPath); err != nil {
		t.Fatal(err)
	}
	c := opt.configCheck(context.Background())
	if c.State != StateFail || c.Hint == "" {
		t.Fatalf("missing config = %+v", c)
	}
}

func TestOptionalServicesAreOffWhenNotConfigured(t *testing.T) {
	opt, _ := newOpts(t, nil)
	opt.Offline = false
	ctx := context.Background()
	for name, c := range map[string]Check{
		"tailscale":     opt.tailscale(ctx),
		"backup_remote": opt.backupRemote(ctx),
		"openrouter":    opt.openRouter(ctx),
	} {
		if c.State != StateOff {
			t.Fatalf("%s = %+v, want off", name, c)
		}
	}
}

func TestBackupRemoteCheck(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedAddr := closed.Addr().String()
	closed.Close()

	opt, _ := newOpts(t, nil)
	opt.Offline = false
	ctx := context.Background()

	opt.Spec.Backup.Destinations = []obelisk.BackupDestination{{ID: "a", Endpoint: "http://" + ln.Addr().String(), Enabled: true}}
	if c := opt.backupRemote(ctx); c.State != StateOK {
		t.Fatalf("listening endpoint = %+v", c)
	}
	opt.Spec.Backup.Destinations = []obelisk.BackupDestination{
		{ID: "a", Endpoint: "http://" + ln.Addr().String(), Enabled: true},
		{ID: "b", Endpoint: "http://" + closedAddr, Enabled: true},
		{ID: "c", Endpoint: "http://unreachable.invalid", Enabled: false},
	}
	c := opt.backupRemote(ctx)
	if c.State != StateFail || c.Detail == "" {
		t.Fatalf("one dead endpoint = %+v", c)
	}
}

func TestCollect(t *testing.T) {
	opt, _ := newOpts(t, nil)
	r := Collect(context.Background(), opt)
	wantOrder := []string{"github_api", "github_oauth", "github_blueprint", "network", "config", "tailscale", "backup_remote", "openrouter"}
	if len(r.Checks) != len(wantOrder) {
		t.Fatalf("got %d checks, want %d", len(r.Checks), len(wantOrder))
	}
	for i, id := range wantOrder {
		if r.Checks[i].ID != id {
			t.Fatalf("check %d = %s, want %s", i, r.Checks[i].ID, id)
		}
	}
	if byID(t, r, "network").State != StateOff {
		t.Fatal("offline must skip network probes")
	}
	if r.CheckedAt == "" {
		t.Fatal("checked_at missing")
	}
}

func TestCollect_githubOnlyOmitsPlatformChecks(t *testing.T) {
	opt, _ := newOpts(t, nil)
	opt.GitHubOnly = true
	for _, c := range Collect(context.Background(), opt).Checks {
		if c.Group != GroupGitHub {
			t.Fatalf("pre-login view leaked a %s check: %s", c.Group, c.ID)
		}
	}
}

func TestOverall(t *testing.T) {
	for _, tt := range []struct {
		states []State
		want   State
	}{
		{[]State{StateOK, StateOff}, StateOK},
		{[]State{StateOK, StateWarn, StateOff}, StateWarn},
		{[]State{StateWarn, StateFail, StateOK}, StateFail},
	} {
		var checks []Check
		for _, s := range tt.states {
			checks = append(checks, Check{State: s})
		}
		if got := overall(checks); got != tt.want {
			t.Fatalf("overall(%v) = %s, want %s", tt.states, got, tt.want)
		}
	}
}

func TestCredentialsUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read everything")
	}
	opt, tokens := newOpts(t, nil)
	if err := tokens.Save(blueprint.Token{AccessToken: "t", Login: "octo"}); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(config.CredentialsDir(opt.ConfigPath), "*.json"))
	if len(files) == 0 {
		t.Fatal("token file not written")
	}
	for _, f := range files {
		if err := os.Chmod(f, 0); err != nil {
			t.Fatal(err)
		}
	}
	c := opt.githubOAuth(context.Background())
	if c.State != StateWarn || c.Hint == "" {
		t.Fatalf("unreadable credentials must warn, not claim signed out: %+v", c)
	}
}
