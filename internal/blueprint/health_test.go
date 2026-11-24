package blueprint

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestClassifyGitHubError(t *testing.T) {
	now := time.Date(2026, 11, 24, 17, 0, 0, 0, time.UTC)
	reset := now.Add(27 * time.Minute)
	tests := []struct {
		name     string
		err      error
		state    string
		blocking bool
	}{
		{"nil is ok", nil, GHStateOK, false},
		{"403 rate limit message", &StatusError{Code: 403, Body: `{"message":"API rate limit exceeded for user ID 1."}`, ResetAt: reset, Remaining: 0}, GHStateRateLimited, true},
		{"403 secondary rate limit", &StatusError{Code: 403, Body: `{"message":"You have exceeded a secondary rate limit."}`, RetryAfter: 60 * time.Second, Remaining: -1}, GHStateRateLimited, true},
		{"403 remaining zero with reset", &StatusError{Code: 403, Body: `{"message":"Forbidden"}`, ResetAt: reset, Remaining: 0}, GHStateRateLimited, true},
		{"429", &StatusError{Code: 429, Remaining: -1}, GHStateRateLimited, true},
		{"403 sso", &StatusError{Code: 403, Body: `{"message":"Resource protected by organization SAML enforcement."}`, SSO: "required; url=https://github.com/orgs/acme/sso?authorization_request=abc", Remaining: -1}, GHStateForbidden, true},
		{"403 permission", &StatusError{Code: 403, Body: `{"message":"Resource not accessible by integration"}`, Remaining: -1}, GHStateForbidden, true},
		{"401", &StatusError{Code: 401, Body: `{"message":"Bad credentials"}`, Remaining: -1}, GHStateUnauthorized, true},
		{"402", &StatusError{Code: 402, Body: `{"message":"Payment required"}`, Remaining: -1}, GHStatePaymentRequired, true},
		{"503", &StatusError{Code: 503, Remaining: -1}, GHStateUnavailable, true},
		{"wrapped status error", fmt.Errorf("fetch: %w", &StatusError{Code: 401, Remaining: -1}), GHStateUnauthorized, true},
		{"404 is not an api health problem", &StatusError{Code: 404, Body: `{"message":"Not Found"}`, Remaining: -1}, GHStateError, false},
		{"422 is not an api health problem", &StatusError{Code: 422, Body: `{"message":"Update is not a fast forward"}`, Remaining: -1}, GHStateError, false},
		{"dial error", &url.Error{Op: "Get", URL: "https://api.github.com", Err: &net.OpError{Op: "dial", Err: errors.New("no route to host")}}, GHStateUnreachable, true},
		{"deadline", context.DeadlineExceeded, GHStateUnreachable, true},
		{"local error", errors.New("no token stored"), GHStateError, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ClassifyGitHubError(tt.err, now)
			if h.State != tt.state {
				t.Fatalf("state = %q, want %q (%+v)", h.State, tt.state, h)
			}
			if h.Blocking != tt.blocking {
				t.Fatalf("blocking = %v, want %v", h.Blocking, tt.blocking)
			}
			if h.Title == "" {
				t.Fatal("title must always be set")
			}
		})
	}
}

func TestClassifyGitHubError_rateLimitDetails(t *testing.T) {
	now := time.Date(2026, 11, 24, 17, 0, 0, 0, time.UTC)
	reset := now.Add(27 * time.Minute)
	rem := 0
	h := ClassifyGitHubError(&StatusError{
		Code: 403, Body: `{"message":"API rate limit exceeded for user ID 42."}`,
		ResetAt: reset, Remaining: rem, RequestID: "ABCD:1234",
	}, now)
	if h.ResetAt != "2026-11-24T17:27:00Z" {
		t.Fatalf("reset_at = %q", h.ResetAt)
	}
	if h.RetryAfterSeconds != 27*60 {
		t.Fatalf("retry_after_seconds = %d, want %d", h.RetryAfterSeconds, 27*60)
	}
	if h.Remaining == nil || *h.Remaining != 0 {
		t.Fatalf("remaining = %v, want 0", h.Remaining)
	}
	if h.RequestID != "ABCD:1234" || h.HTTPStatus != 403 {
		t.Fatalf("request id / status lost: %+v", h)
	}
	if !strings.Contains(h.Message, "rate limit exceeded") {
		t.Fatalf("message = %q", h.Message)
	}
}

func TestClassifyGitHubError_ssoURL(t *testing.T) {
	h := ClassifyGitHubError(&StatusError{
		Code: 403, SSO: "required; url=https://github.com/orgs/acme/sso?authorization_request=abc", Remaining: -1,
	}, time.Now())
	if h.SSOURL != "https://github.com/orgs/acme/sso?authorization_request=abc" {
		t.Fatalf("sso_url = %q", h.SSOURL)
	}
}

func TestNewStatusError_readsHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	rr.Header().Set("X-RateLimit-Remaining", "0")
	rr.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Date(2026, 11, 24, 17, 27, 27, 0, time.UTC).Unix(), 10))
	rr.Header().Set("Retry-After", "90")
	rr.Header().Set("X-GitHub-Request-Id", "REQ1")
	rr.WriteHeader(403)
	se := newStatusError(rr.Result(), `{"message":"x"}`)
	if se.Remaining != 0 || se.RetryAfter != 90*time.Second || se.RequestID != "REQ1" {
		t.Fatalf("headers not parsed: %+v", se)
	}
	if !se.ResetAt.Equal(time.Date(2026, 11, 24, 17, 27, 27, 0, time.UTC)) {
		t.Fatalf("reset = %v", se.ResetAt)
	}
}

func githubServer(t *testing.T, h http.HandlerFunc) *GitHub {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &GitHub{
		APIBase: srv.URL,
		Client:  srv.Client(),
		Token:   func(context.Context) (string, error) { return "tok", nil },
	}
}

func TestCheckAPI_ok(t *testing.T) {
	g := githubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rate_limit" {
			t.Errorf("path = %s, want /rate_limit (the only unmetered endpoint)", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"resources":{"core":{"limit":5000,"remaining":4321,"reset":1795000000}}}`))
	})
	h := g.CheckAPI(context.Background())
	if h.State != GHStateOK || h.Blocking {
		t.Fatalf("health = %+v", h)
	}
	if h.Limit != 5000 || h.Remaining == nil || *h.Remaining != 4321 || !h.TokenPresent {
		t.Fatalf("quota not reported: %+v", h)
	}
}

func TestCheckAPI_quotaExhausted(t *testing.T) {
	reset := time.Now().Add(20 * time.Minute).Unix()
	g := githubServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"resources":{"core":{"limit":5000,"remaining":0,"reset":%d}}}`, reset)
	})
	h := g.CheckAPI(context.Background())
	if h.State != GHStateRateLimited || !h.Blocking {
		t.Fatalf("health = %+v, want rate_limited", h)
	}
	if h.Remaining == nil || *h.Remaining != 0 || h.Limit != 5000 {
		t.Fatalf("quota = %+v", h)
	}
	if h.ResetAt == "" || h.RetryAfterSeconds <= 0 {
		t.Fatalf("reset info missing: %+v", h)
	}
}

func TestCheckAPI_statusErrors(t *testing.T) {
	for code, want := range map[int]string{
		http.StatusUnauthorized:    GHStateUnauthorized,
		http.StatusPaymentRequired: GHStatePaymentRequired,
		http.StatusForbidden:       GHStateForbidden,
		http.StatusBadGateway:      GHStateUnavailable,
	} {
		g := githubServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"message":"nope"}`))
		})
		h := g.CheckAPI(context.Background())
		if h.State != want || !h.Blocking || h.HTTPStatus != code {
			t.Fatalf("HTTP %d -> %+v, want state %s", code, h, want)
		}
	}
}

func TestCheckAPI_unreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	g := &GitHub{APIBase: srv.URL, Client: srv.Client()}
	srv.Close()
	h := g.CheckAPI(context.Background())
	if h.State != GHStateUnreachable || !h.Blocking {
		t.Fatalf("health = %+v, want unreachable", h)
	}
}

func TestCheckAPI_anonymousWithoutToken(t *testing.T) {
	var auth string
	g := githubServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"resources":{"core":{"limit":60,"remaining":60,"reset":0}}}`))
	})
	g.Token = func(context.Context) (string, error) { return "", errors.New("no token") }
	h := g.CheckAPI(context.Background())
	if h.State != GHStateOK || h.TokenPresent || auth != "" {
		t.Fatalf("health = %+v auth=%q, want anonymous ok", h, auth)
	}
}

// newStatusRuntime builds a runtime with a GitHub fake. repo is the blueprint
// repository in the config ("" = never onboarded).
func newStatusRuntime(t *testing.T, repo string, h http.HandlerFunc) *Runtime {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-status01"
	spec.Blueprint.GitHub.Repository = repo
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}
	rt, err := NewRuntime(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Tokens.Save(Token{AccessToken: "tok", Login: "octo", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	rt.Git = githubServer(t, h)
	rt.Git.Token = rt.AccessToken
	return rt
}

// The reported bug: a GitHub rate limit made the console fall back to the
// "new appliance" onboarding screen.
func TestStatus_rateLimitIsNotOnboarding(t *testing.T) {
	reset := time.Now().Add(20 * time.Minute).Unix()
	rt := newStatusRuntime(t, "", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded for user ID 42."}`))
	})
	st := rt.Status(context.Background(), "octo")
	if st.NeedsOnboarding {
		t.Fatal("needs_onboarding must be false while GitHub is rate limiting")
	}
	if st.GitHubAPI.State != GHStateRateLimited || !st.GitHubAPI.Blocking {
		t.Fatalf("github_api = %+v", st.GitHubAPI)
	}
	if st.GitHubAPI.ResetAt == "" {
		t.Fatal("reset time missing")
	}
}

func TestStatus_remoteForbiddenIsNotOnboarding(t *testing.T) {
	rt := newStatusRuntime(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rate_limit" {
			_, _ = w.Write([]byte(`{"resources":{"core":{"limit":5000,"remaining":4000,"reset":0}}}`))
			return
		}
		w.Header().Set("X-GitHub-SSO", "required; url=https://github.com/orgs/acme/sso?authorization_request=x")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Resource protected by organization SAML enforcement."}`))
	})
	st := rt.Status(context.Background(), "octo")
	if st.NeedsOnboarding {
		t.Fatal("needs_onboarding must be false when the blueprint repo cannot be read")
	}
	if !st.RemoteCheckFailed {
		t.Fatal("remote_check_failed should be set")
	}
	if st.GitHubAPI.State != GHStateForbidden || st.GitHubAPI.SSOURL == "" {
		t.Fatalf("github_api = %+v", st.GitHubAPI)
	}
}

// Guard: a healthy GitHub with no blueprint repo is still a new appliance.
func TestStatus_noRepoOnHealthyGitHubNeedsOnboarding(t *testing.T) {
	rt := newStatusRuntime(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rate_limit" {
			_, _ = w.Write([]byte(`{"resources":{"core":{"limit":5000,"remaining":4000,"reset":0}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	})
	st := rt.Status(context.Background(), "octo")
	if !st.NeedsOnboarding {
		t.Fatalf("expected onboarding on a healthy GitHub with no repo: %+v", st)
	}
	if st.GitHubAPI.State != GHStateOK || st.RemoteCheckFailed {
		t.Fatalf("github_api = %+v remote_check_failed=%v", st.GitHubAPI, st.RemoteCheckFailed)
	}
}

func TestStatus_blockedGitHubSkipsRepoProbe(t *testing.T) {
	var calls int
	rt := newStatusRuntime(t, "octo/obelisk-appliance", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"message":"Payment required"}`))
	})
	st := rt.Status(context.Background(), "octo")
	if st.GitHubAPI.State != GHStatePaymentRequired {
		t.Fatalf("github_api = %+v", st.GitHubAPI)
	}
	if calls != 1 {
		t.Fatalf("GitHub calls = %d, want 1 (only the free health probe)", calls)
	}
}

func TestNoteGitHubError_overridesHealthyProbe(t *testing.T) {
	rt := newStatusRuntime(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"resources":{"core":{"limit":5000,"remaining":4000,"reset":0}}}`))
	})
	rt.NoteGitHubError(&StatusError{Code: 403, SSO: "required; url=https://github.com/orgs/acme/sso", Remaining: -1})
	h := rt.GitHubAPIHealth(context.Background(), false)
	if h.State != GHStateForbidden || h.SSOURL == "" {
		t.Fatalf("noted sign-in failure should surface: %+v", h)
	}
	rt.ClearGitHubError()
	if h := rt.GitHubAPIHealth(context.Background(), false); h.State != GHStateOK {
		t.Fatalf("after clear: %+v", h)
	}
	rt.NoteGitHubError(&StatusError{Code: 404, Remaining: -1})
	if h := rt.GitHubAPIHealth(context.Background(), false); h.State != GHStateOK {
		t.Fatalf("non-blocking errors must not be noted: %+v", h)
	}
}
