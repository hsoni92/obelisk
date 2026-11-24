package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/syshealth"
)

// A GitHub failure must reach the UI as a classified report, not a generic
// error, so the console can show the real reason instead of onboarding.
func TestWriteBlueprintErr_classifiesGitHubFailures(t *testing.T) {
	reset := time.Now().Add(20 * time.Minute)
	tests := []struct {
		name       string
		err        error
		wantHTTP   int
		wantCode   string
		retryAfter bool
	}{
		{"rate limit", &blueprint.StatusError{Code: 403, Body: `{"message":"API rate limit exceeded"}`, ResetAt: reset, Remaining: 0}, http.StatusTooManyRequests, "github_rate_limited", true},
		{"forbidden", &blueprint.StatusError{Code: 403, Body: `{"message":"Resource not accessible"}`, Remaining: -1}, http.StatusForbidden, "github_forbidden", false},
		{"payment required", &blueprint.StatusError{Code: 402, Body: `{"message":"Payment required"}`, Remaining: -1}, http.StatusPaymentRequired, "github_payment_required", false},
		// 401 from GitHub must not become our 401, which means "admin sign-in required".
		{"upstream 401", &blueprint.StatusError{Code: 401, Body: `{"message":"Bad credentials"}`, Remaining: -1}, http.StatusBadGateway, "github_unauthorized", false},
		{"outage", &blueprint.StatusError{Code: 503, Remaining: -1}, http.StatusBadGateway, "github_unavailable", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeBlueprintErr(rr, tt.err)
			if rr.Code != tt.wantHTTP {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantHTTP, rr.Body.String())
			}
			var body struct {
				Error  string                 `json:"error"`
				Code   string                 `json:"code"`
				Hint   string                 `json:"hint"`
				GitHub blueprint.GitHubHealth `json:"github"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tt.wantCode || body.GitHub.State == "" || !body.GitHub.Blocking {
				t.Fatalf("body = %+v", body)
			}
			if body.Error == "" || body.Hint == "" {
				t.Fatalf("error and hint must be set for the toast: %+v", body)
			}
			if got := rr.Header().Get("Retry-After") != ""; got != tt.retryAfter {
				t.Fatalf("Retry-After present = %v, want %v", got, tt.retryAfter)
			}
		})
	}
}

func TestWriteBlueprintErr_nonGitHubErrorsKeepOldBehavior(t *testing.T) {
	rr := httptest.NewRecorder()
	writeBlueprintErr(rr, &blueprint.StatusError{Code: 404, Body: `{"message":"Not Found"}`, Remaining: -1})
	if rr.Code == http.StatusBadGateway || rr.Code == http.StatusTooManyRequests {
		t.Fatalf("a 404 is not a GitHub outage; status = %d", rr.Code)
	}
}

func TestRedactPublicReport_hidesIdentities(t *testing.T) {
	in := syshealth.Report{Checks: []syshealth.Check{
		{ID: "github_api", State: syshealth.StateOK, Summary: "reachable"},
		{ID: "github_oauth", State: syshealth.StateOK, Summary: "signed in as octo", Detail: "scopes repo"},
		{ID: "github_blueprint", State: syshealth.StateWarn, Summary: "synced · octo/obelisk-appliance@machine-x · gen 4", Detail: "last error mentions octo/obelisk-appliance"},
	}}
	out := redactPublicReport(in)
	for _, c := range out.Checks {
		for _, leak := range []string{"octo", "obelisk-appliance", "machine-x"} {
			if strings.Contains(c.Summary, leak) || strings.Contains(c.Detail, leak) {
				t.Fatalf("%s leaks %q: %+v", c.ID, leak, c)
			}
		}
	}
	if out.Checks[1].Summary != "signed in" {
		t.Fatalf("oauth summary = %q", out.Checks[1].Summary)
	}
	// The input must not be mutated: the authenticated view shares the slice source.
	if in.Checks[1].Summary != "signed in as octo" {
		t.Fatal("redaction mutated its input")
	}
}

func TestSystemHealthIsPublic(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	if !isPublic(r) {
		t.Fatal("system health must be reachable before sign-in so the error screen can explain GitHub failures")
	}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/system/health", nil)
	if isPublic(r) {
		t.Fatal("only GET is public")
	}
}
