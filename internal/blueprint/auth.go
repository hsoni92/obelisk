package blueprint

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
)

// DefaultGitHubOAuthScope is requested on authorize (repo for git push; read:user for /user login lookup).
const DefaultGitHubOAuthScope = "repo read:user"

// OAuthConfig is the GitHub OAuth App used for login and git transport.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scope        string
}

func (c OAuthConfig) Configured() bool {
	return strings.TrimSpace(c.ClientID) != ""
}

// LoadOAuthConfig reads the OAuth app from the environment, then a mode-0600 file
// beside credentials: github-oauth.client.json {"client_id","client_secret","redirect_uri"}.
func LoadOAuthConfig(configPath string) OAuthConfig {
	cfg := OAuthConfig{
		ClientID:     strings.TrimSpace(os.Getenv("OBELISK_GITHUB_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("OBELISK_GITHUB_CLIENT_SECRET")),
		RedirectURI:  strings.TrimSpace(os.Getenv("OBELISK_GITHUB_REDIRECT_URI")),
		Scope:        DefaultGitHubOAuthScope,
	}
	path := filepath.Join(config.CredentialsDir(configPath), "github-oauth.client.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	var file struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
	}
	if json.Unmarshal(data, &file) != nil {
		return cfg
	}
	if cfg.ClientID == "" {
		cfg.ClientID = strings.TrimSpace(file.ClientID)
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = strings.TrimSpace(file.ClientSecret)
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = strings.TrimSpace(file.RedirectURI)
	}
	return cfg
}

// AuthorizeStart is the payload the admin UI needs to send the operator to GitHub.
type AuthorizeStart struct {
	Flow            string `json:"flow"`
	AuthorizeURL    string `json:"authorize_url,omitempty"`
	State           string `json:"state,omitempty"`
	UserCode        string `json:"user_code,omitempty"`
	VerificationURI string `json:"verification_uri,omitempty"`
	DeviceCode      string `json:"device_code,omitempty"`
	Interval        int    `json:"interval,omitempty"`
	ExpiresIn       int    `json:"expires_in,omitempty"`
}

type pendingAuth struct {
	flow        string
	verifier    string
	redirectURI string
	deviceCode  string
	created     time.Time
}

// OAuth talks to GitHub's authorization and device endpoints.
type OAuth struct {
	Config          OAuthConfig
	Client          *http.Client
	AuthBase        string
	CredentialsDir  string
	mu              sync.Mutex
	pending         map[string]pendingAuth
}

func NewOAuth(cfg OAuthConfig) *OAuth {
	return &OAuth{
		Config:   cfg,
		AuthBase: "https://github.com",
		pending:  map[string]pendingAuth{},
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// StartCode begins Authorization Code + PKCE. redirectURI may be supplied by the UI;
// GitHub still enforces the OAuth app allowlist.
func (o *OAuth) StartCode(redirectURI string) (AuthorizeStart, error) {
	if !o.Config.Configured() {
		return AuthorizeStart{}, ErrClientNotConfigured
	}
	if redirectURI == "" {
		redirectURI = o.Config.RedirectURI
	}
	if redirectURI == "" {
		return AuthorizeStart{}, fmt.Errorf("redirect_uri is required")
	}
	verifier, challenge, err := pkcePair()
	if err != nil {
		return AuthorizeStart{}, err
	}
	state, err := randomToken(24)
	if err != nil {
		return AuthorizeStart{}, err
	}
	p := pendingAuth{flow: "code", verifier: verifier, redirectURI: redirectURI, created: time.Now()}
	o.mu.Lock()
	o.sweepLocked()
	o.pending[state] = p
	o.mu.Unlock()
	_ = o.writePendingFile(state, p)

	q := url.Values{}
	q.Set("client_id", o.Config.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", o.Config.Scope)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return AuthorizeStart{
		Flow:         "code",
		AuthorizeURL: o.AuthBase + "/login/oauth/authorize?" + q.Encode(),
		State:        state,
	}, nil
}

// StartDevice begins the device authorization grant for a headless VPS.
func (o *OAuth) StartDevice(ctx context.Context) (AuthorizeStart, error) {
	if !o.Config.Configured() {
		return AuthorizeStart{}, ErrClientNotConfigured
	}
	form := url.Values{}
	form.Set("client_id", o.Config.ClientID)
	form.Set("scope", o.Config.Scope)
	var out struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
		ErrorDesc       string `json:"error_description"`
	}
	if err := o.postForm(ctx, o.AuthBase+"/login/device/code", form, &out); err != nil {
		return AuthorizeStart{}, err
	}
	if out.Error != "" {
		return AuthorizeStart{}, fmt.Errorf("github device: %s", out.ErrorDesc)
	}
	o.mu.Lock()
	o.sweepLocked()
	o.pending[out.DeviceCode] = pendingAuth{flow: "device", deviceCode: out.DeviceCode, created: time.Now()}
	o.mu.Unlock()
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return AuthorizeStart{
		Flow:            "device",
		UserCode:        out.UserCode,
		VerificationURI: out.VerificationURI,
		DeviceCode:      out.DeviceCode,
		Interval:        out.Interval,
		ExpiresIn:       out.ExpiresIn,
	}, nil
}

// ExchangeCode completes the browser flow.
func (o *OAuth) ExchangeCode(ctx context.Context, code, state string) (Token, error) {
	p, ok := o.lookupPendingCode(state)
	if !ok || p.flow != "code" || time.Since(p.created) > 10*time.Minute {
		return Token{}, fmt.Errorf("oauth state expired")
	}
	form := url.Values{}
	form.Set("client_id", o.Config.ClientID)
	form.Set("client_secret", o.Config.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", p.redirectURI)
	form.Set("code_verifier", p.verifier)
	form.Set("grant_type", "authorization_code")
	tok, err := o.tokenFromForm(ctx, form)
	if err != nil {
		return Token{}, err
	}
	o.forgetPending(state)
	return tok, nil
}

// PollDevice checks a device grant. ErrPending means the operator has not approved yet.
func (o *OAuth) PollDevice(ctx context.Context, deviceCode string) (Token, error) {
	o.mu.Lock()
	p, ok := o.pending[deviceCode]
	o.mu.Unlock()
	if !ok || p.flow != "device" {
		return Token{}, fmt.Errorf("unknown device code")
	}
	form := url.Values{}
	form.Set("client_id", o.Config.ClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	tok, err := o.tokenFromForm(ctx, form)
	if err != nil {
		return Token{}, err
	}
	o.mu.Lock()
	delete(o.pending, deviceCode)
	o.mu.Unlock()
	return tok, nil
}

// Refresh exchanges a refresh token for a new access token.
func (o *OAuth) Refresh(ctx context.Context, current Token) (Token, error) {
	if current.RefreshToken == "" {
		return current, nil
	}
	form := url.Values{}
	form.Set("client_id", o.Config.ClientID)
	form.Set("client_secret", o.Config.ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", current.RefreshToken)
	tok, err := o.tokenFromForm(ctx, form)
	if err != nil {
		return Token{}, err
	}
	if tok.RefreshToken == "" {
		tok.RefreshToken = current.RefreshToken
	}
	if tok.Login == "" {
		tok.Login = current.Login
	}
	return tok, nil
}

func (o *OAuth) tokenFromForm(ctx context.Context, form url.Values) (Token, error) {
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := o.postForm(ctx, o.AuthBase+"/login/oauth/access_token", form, &out); err != nil {
		return Token{}, err
	}
	switch out.Error {
	case "":
	case "authorization_pending", "slow_down":
		return Token{}, ErrPending
	default:
		msg := out.ErrorDesc
		if msg == "" {
			msg = out.Error
		}
		return Token{}, fmt.Errorf("github oauth: %s", msg)
	}
	if out.AccessToken == "" {
		return Token{}, fmt.Errorf("github oauth: empty access token")
	}
	tok := Token{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		TokenType:    out.TokenType,
		Scope:        out.Scope,
	}
	if out.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	}
	return tok, nil
}

func (o *OAuth) postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := o.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		msg := fmt.Sprintf("github oauth http %d", resp.StatusCode)
		if b := strings.TrimSpace(string(body)); b != "" {
			var er struct {
				Error            string `json:"error"`
				ErrorDescription string `json:"error_description"`
			}
			if json.Unmarshal(body, &er) == nil {
				if er.ErrorDescription != "" {
					msg += ": " + er.ErrorDescription
				} else if er.Error != "" {
					msg += ": " + er.Error
				}
			} else if len(b) < 200 {
				msg += ": " + b
			}
		}
		return fmt.Errorf("%s", msg)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("github oauth response: %w", err)
	}
	return nil
}

func (o *OAuth) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return http.DefaultClient
}

func (o *OAuth) sweepLocked() {
	for k, v := range o.pending {
		if time.Since(v.created) > 15*time.Minute {
			delete(o.pending, k)
			o.removePendingFile(k)
		}
	}
	o.sweepPendingFiles()
}

type pendingAuthRecord struct {
	Flow        string    `json:"flow"`
	Verifier    string    `json:"verifier,omitempty"`
	RedirectURI string    `json:"redirect_uri,omitempty"`
	Created     time.Time `json:"created"`
}

func (o *OAuth) pendingFilePath(state string) string {
	if o.CredentialsDir == "" || state == "" {
		return ""
	}
	return filepath.Join(o.CredentialsDir, "oauth-pending-"+state+".json")
}

func (o *OAuth) writePendingFile(state string, p pendingAuth) error {
	path := o.pendingFilePath(state)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	rec := pendingAuthRecord{Flow: p.flow, Verifier: p.verifier, RedirectURI: p.redirectURI, Created: p.created}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (o *OAuth) readPendingFile(state string) (pendingAuth, bool) {
	path := o.pendingFilePath(state)
	if path == "" {
		return pendingAuth{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return pendingAuth{}, false
	}
	var rec pendingAuthRecord
	if json.Unmarshal(data, &rec) != nil || rec.Flow != "code" {
		return pendingAuth{}, false
	}
	return pendingAuth{
		flow:        rec.Flow,
		verifier:    rec.Verifier,
		redirectURI: rec.RedirectURI,
		created:     rec.Created,
	}, true
}

func (o *OAuth) removePendingFile(state string) {
	path := o.pendingFilePath(state)
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

func (o *OAuth) lookupPendingCode(state string) (pendingAuth, bool) {
	o.mu.Lock()
	p, ok := o.pending[state]
	o.mu.Unlock()
	if ok {
		return p, true
	}
	return o.readPendingFile(state)
}

func (o *OAuth) forgetPending(state string) {
	o.mu.Lock()
	delete(o.pending, state)
	o.mu.Unlock()
	o.removePendingFile(state)
}

func (o *OAuth) sweepPendingFiles() {
	dir := o.CredentialsDir
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-15 * time.Minute)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "oauth-pending-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

func pkcePair() (verifier, challenge string, err error) {
	verifier, err = randomToken(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
