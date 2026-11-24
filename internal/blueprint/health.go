package blueprint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GitHub API health states reported to the admin UI and `obelisk status`.
const (
	GHStateOK              = "ok"
	GHStateRateLimited     = "rate_limited"
	GHStateForbidden       = "forbidden"
	GHStateUnauthorized    = "unauthorized"
	GHStatePaymentRequired = "payment_required"
	GHStateUnavailable     = "unavailable"
	GHStateUnreachable     = "unreachable"
	GHStateError           = "error"
)

const (
	githubHealthTTL        = 30 * time.Second
	githubHealthFreshFloor = 3 * time.Second
	githubNotedErrorTTL    = 10 * time.Minute
	githubProbeTimeout     = 6 * time.Second
)

// GitHubHealth explains whether the GitHub API can be used right now and, when
// it cannot, why and what to do. Blocking means sign-in, onboarding and
// blueprint sync cannot work, so the UI must not fall back to "new appliance"
// flows that would only fail again.
type GitHubHealth struct {
	State             string   `json:"state"`
	Blocking          bool     `json:"blocking"`
	HTTPStatus        int      `json:"http_status,omitempty"`
	Title             string   `json:"title"`
	Message           string   `json:"message,omitempty"`
	Hint              string   `json:"hint,omitempty"`
	Steps             []string `json:"steps,omitempty"`
	ResetAt           string   `json:"reset_at,omitempty"`
	RetryAfterSeconds int      `json:"retry_after_seconds,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	Remaining         *int     `json:"remaining,omitempty"`
	SSOURL            string   `json:"sso_url,omitempty"`
	RequestID         string   `json:"request_id,omitempty"`
	TokenPresent      bool     `json:"token_present"`
	LatencyMS         int64    `json:"latency_ms,omitempty"`
	CheckedAt         string   `json:"checked_at"`
}

var ssoURLRE = regexp.MustCompile(`url=(\S+)`)

// newStatusError builds a StatusError carrying the rate-limit, SSO and request
// id headers GitHub sends with failures.
func newStatusError(resp *http.Response, body string) *StatusError {
	se := &StatusError{
		Code:      resp.StatusCode,
		Body:      sanitize(body),
		Remaining: -1,
		SSO:       resp.Header.Get("X-GitHub-SSO"),
		RequestID: resp.Header.Get("X-GitHub-Request-Id"),
	}
	if v := resp.Header.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			se.Remaining = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			se.ResetAt = time.Unix(n, 0).UTC()
		}
	}
	if v := resp.Header.Get("Retry-After"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			se.RetryAfter = time.Duration(n) * time.Second
		}
	}
	return se
}

// message returns GitHub's human message without the documentation URL.
func (e *StatusError) message() string {
	var body struct {
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(e.Body), &body) == nil && body.Message != "" {
		return strings.TrimSpace(body.Message)
	}
	return strings.TrimSpace(e.Body)
}

// IsRateLimit reports a 429, a secondary rate limit, or a 403 caused by the
// primary rate limit.
func (e *StatusError) IsRateLimit() bool {
	if e == nil {
		return false
	}
	if e.Code == http.StatusTooManyRequests {
		return true
	}
	if e.Code != http.StatusForbidden {
		return false
	}
	msg := strings.ToLower(e.message())
	if strings.Contains(msg, "rate limit") || strings.Contains(msg, "abuse detection") {
		return true
	}
	return e.Remaining == 0 && !e.ResetAt.IsZero()
}

// ClassifyGitHubError turns an error from a GitHub call into a health report.
// A nil error is healthy. Errors that are not about reaching or being allowed
// to use the API (for example a missing local token) come back as
// GHStateError with Blocking false.
func ClassifyGitHubError(err error, now time.Time) GitHubHealth {
	h := GitHubHealth{CheckedAt: now.UTC().Format(time.RFC3339)}
	if err == nil {
		h.State = GHStateOK
		h.Title = "GitHub API is reachable"
		return h
	}
	var se *StatusError
	if errors.As(err, &se) {
		return classifyStatusError(se, now, h)
	}
	if isNetworkError(err) {
		h.State = GHStateUnreachable
		h.Blocking = true
		h.Title = "Cannot reach GitHub"
		h.Message = "The appliance could not open a connection to api.github.com."
		h.Hint = "Sign-in and blueprint sync resume automatically once the network path is back."
		h.Steps = []string{
			"On the appliance run: curl -sSI https://api.github.com",
			"Check DNS, the outbound firewall and any HTTP proxy for HTTPS (443).",
			"Confirm the system clock is correct; TLS fails when it drifts.",
		}
		return h
	}
	h.State = GHStateError
	h.Title = "GitHub request failed"
	h.Message = sanitize(err.Error())
	return h
}

func classifyStatusError(se *StatusError, now time.Time, h GitHubHealth) GitHubHealth {
	h.HTTPStatus = se.Code
	h.Message = se.message()
	h.RequestID = se.RequestID
	switch {
	case se.IsRateLimit():
		h.State = GHStateRateLimited
		h.Blocking = true
		h.Title = "GitHub API rate limit reached"
		reset := se.ResetAt
		if reset.IsZero() && se.RetryAfter > 0 {
			reset = now.Add(se.RetryAfter)
		}
		if !reset.IsZero() {
			h.ResetAt = reset.UTC().Format(time.RFC3339)
			if wait := int(reset.Sub(now).Seconds()); wait > 0 {
				h.RetryAfterSeconds = wait
			}
		}
		if se.Remaining >= 0 {
			rem := se.Remaining
			h.Remaining = &rem
		}
		h.Hint = "Nothing is wrong with your account. Sign-in, onboarding and blueprint sync pause automatically and resume when the limit resets."
		h.Steps = []string{
			"Wait for the reset time shown here; no action is needed on your side.",
			"Stop other tools or scripts that use the same GitHub account or OAuth token.",
			"Automatic blueprint pushes are rate limited; use \"Push now\" sparingly until it resets.",
		}
	case se.Code == http.StatusUnauthorized:
		h.State = GHStateUnauthorized
		h.Blocking = true
		h.Title = "GitHub rejected the stored sign-in"
		h.Hint = "The saved OAuth token is expired, revoked, or no longer valid for this OAuth app."
		h.Steps = []string{
			"Sign in with GitHub again to issue a new token.",
			"If it keeps failing, check github.com → Settings → Applications → Authorized OAuth Apps and re-authorize Obelisk.",
			"Verify OBELISK_GITHUB_CLIENT_ID and the client secret match the OAuth app (a rotated secret breaks token refresh).",
		}
	case se.Code == http.StatusPaymentRequired:
		h.State = GHStatePaymentRequired
		h.Blocking = true
		h.Title = "GitHub billing or plan limit"
		h.Hint = "GitHub refused the request because of the account's plan, billing state, or a spending limit."
		h.Steps = []string{
			"Open github.com → Settings → Billing and plans and resolve any payment or spending-limit issue.",
			"Check that the account can create or use the private blueprint repository on its current plan.",
			"Retry here after the account is in good standing.",
		}
	case se.Code == http.StatusForbidden && se.SSO != "":
		h.State = GHStateForbidden
		h.Blocking = true
		h.Title = "Organization SSO authorization required"
		if m := ssoURLRE.FindStringSubmatch(se.SSO); len(m) == 2 {
			h.SSOURL = strings.TrimRight(m[1], ";,")
		}
		h.Hint = "The repository belongs to an organization that enforces SAML SSO for OAuth apps."
		h.Steps = []string{
			"Open the SSO link and authorize Obelisk for the organization.",
			"Then retry here; no new sign-in is needed.",
		}
	case se.Code == http.StatusForbidden:
		h.State = GHStateForbidden
		h.Blocking = true
		h.Title = "GitHub denied access"
		h.Hint = "GitHub understood the request but refused it for this account or OAuth app."
		h.Steps = []string{
			"Check github.com → Settings → Applications → Authorized OAuth Apps: Obelisk needs the repo and read:user scopes.",
			"If the repo is in an organization, ask an owner to approve the app under OAuth app access restrictions.",
			"Make sure the account still has access to the blueprint repository.",
			"Sign out and sign in again so the token picks up the current scopes.",
		}
	case se.Code >= 500:
		h.State = GHStateUnavailable
		h.Blocking = true
		h.Title = "GitHub is having problems"
		h.Hint = "GitHub returned a server error. This is not caused by your appliance."
		h.Steps = []string{
			"Check githubstatus.com for an active incident.",
			"Obelisk retries automatically; sign-in resumes when GitHub recovers.",
		}
	default:
		h.State = GHStateError
		h.Title = fmt.Sprintf("Unexpected GitHub response (HTTP %d)", se.Code)
	}
	return h
}

func isNetworkError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	var ue *url.Error
	return errors.As(err, &ue)
}

// CheckAPI probes GET /rate_limit, which GitHub does not count against the
// quota, so it is safe to call repeatedly. With a stored token it reports that
// account's quota; without one it checks anonymous reachability only.
func (g *GitHub) CheckAPI(ctx context.Context) GitHubHealth {
	start := time.Now()
	token := ""
	if g.Token != nil {
		if t, err := g.Token(ctx); err == nil {
			token = t
		}
	}
	finish := func(h GitHubHealth) GitHubHealth {
		h.TokenPresent = token != ""
		h.LatencyMS = time.Since(start).Milliseconds()
		return h
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.api()+"/rate_limit", nil)
	if err != nil {
		return finish(ClassifyGitHubError(err, time.Now()))
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return finish(ClassifyGitHubError(err, time.Now()))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return finish(ClassifyGitHubError(err, time.Now()))
	}
	if resp.StatusCode >= 300 {
		return finish(ClassifyGitHubError(newStatusError(resp, string(body)), time.Now()))
	}
	var rl struct {
		Resources struct {
			Core struct {
				Limit     int   `json:"limit"`
				Remaining int   `json:"remaining"`
				Reset     int64 `json:"reset"`
			} `json:"core"`
		} `json:"resources"`
	}
	_ = json.Unmarshal(body, &rl)
	core := rl.Resources.Core
	now := time.Now()
	h := ClassifyGitHubError(nil, now)
	h.Limit = core.Limit
	rem := core.Remaining
	h.Remaining = &rem
	if core.Reset > 0 {
		h.ResetAt = time.Unix(core.Reset, 0).UTC().Format(time.RFC3339)
	}
	if core.Limit > 0 && core.Remaining <= 0 {
		synthetic := &StatusError{
			Code:      http.StatusForbidden,
			Body:      fmt.Sprintf(`{"message":"API rate limit exceeded: 0 of %d requests remain."}`, core.Limit),
			Remaining: 0,
			ResetAt:   time.Unix(core.Reset, 0).UTC(),
		}
		limited := classifyStatusError(synthetic, now, GitHubHealth{CheckedAt: now.UTC().Format(time.RFC3339)})
		limited.Limit = core.Limit
		limited.Remaining = &rem
		return finish(limited)
	}
	if token == "" {
		h.Message = "Checked anonymously; no GitHub sign-in is stored on this appliance yet."
	}
	return finish(h)
}

// GitHubAPIHealth returns the cached GitHub API health, probing at most every 30s.
// fresh asks for a new probe (still throttled to once per few seconds). A recent
// blocking failure from a real operation, such as an SSO 403 on sign-in that
// /rate_limit cannot see, overrides a healthy probe until it ages out.
func (rt *Runtime) GitHubAPIHealth(ctx context.Context, fresh bool) GitHubHealth {
	if rt == nil || rt.Git == nil {
		return ClassifyGitHubError(nil, time.Now())
	}
	rt.healthProbeMu.Lock()
	rt.healthMu.Lock()
	cached, at := rt.ghHealth, rt.ghHealthAt
	rt.healthMu.Unlock()
	age := time.Since(at)
	if at.IsZero() || age > githubHealthTTL || (fresh && age > githubHealthFreshFloor) {
		pctx, cancel := context.WithTimeout(ctx, githubProbeTimeout)
		cached = rt.Git.CheckAPI(pctx)
		cancel()
		rt.healthMu.Lock()
		rt.ghHealth, rt.ghHealthAt = cached, time.Now()
		rt.healthMu.Unlock()
	}
	rt.healthProbeMu.Unlock()

	rt.healthMu.Lock()
	noted, notedAt := rt.ghNoted, rt.ghNotedAt
	rt.healthMu.Unlock()
	if noted != nil && time.Since(notedAt) < githubNotedErrorTTL && !cached.Blocking {
		out := *noted
		out.TokenPresent = cached.TokenPresent
		return out
	}
	return cached
}

// NoteGitHubError records a blocking failure seen during a real operation.
func (rt *Runtime) NoteGitHubError(err error) {
	if rt == nil || err == nil {
		return
	}
	h := ClassifyGitHubError(err, time.Now())
	if !h.Blocking {
		return
	}
	rt.healthMu.Lock()
	rt.ghNoted, rt.ghNotedAt = &h, time.Now()
	rt.healthMu.Unlock()
}

// ClearGitHubError drops a noted failure after the same kind of call succeeds.
func (rt *Runtime) ClearGitHubError() {
	if rt == nil {
		return
	}
	rt.healthMu.Lock()
	rt.ghNoted = nil
	rt.healthMu.Unlock()
}
