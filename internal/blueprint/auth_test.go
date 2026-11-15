package blueprint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOAuthCodeExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("code") != "abc" || r.Form.Get("code_verifier") == "" {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "gho_test",
			"token_type":   "bearer",
			"scope":        "repo",
		})
	}))
	defer srv.Close()

	o := NewOAuth(OAuthConfig{ClientID: "id", ClientSecret: "secret", Scope: "repo"})
	o.AuthBase = srv.URL
	o.Client = srv.Client()
	start, err := o.StartCode(srv.URL + "/login/callback")
	if err != nil {
		t.Fatal(err)
	}
	tok, err := o.ExchangeCode(context.Background(), "abc", start.State)
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "gho_test" {
		t.Fatalf("token = %+v", tok)
	}
}

func TestOAuthCodeExchange_PendingSurvivesRestart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "gho_test",
			"token_type":   "bearer",
			"scope":        "repo",
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	o := NewOAuth(OAuthConfig{ClientID: "id", ClientSecret: "secret", Scope: "repo"})
	o.AuthBase = srv.URL
	o.Client = srv.Client()
	o.CredentialsDir = dir
	start, err := o.StartCode(srv.URL + "/login/callback")
	if err != nil {
		t.Fatal(err)
	}
	o.mu.Lock()
	delete(o.pending, start.State)
	o.mu.Unlock()

	o2 := NewOAuth(OAuthConfig{ClientID: "id", ClientSecret: "secret", Scope: "repo"})
	o2.AuthBase = srv.URL
	o2.Client = srv.Client()
	o2.CredentialsDir = dir
	tok, err := o2.ExchangeCode(context.Background(), "abc", start.State)
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "gho_test" {
		t.Fatalf("token = %+v", tok)
	}
	if _, err := os.Stat(filepath.Join(dir, "oauth-pending-"+start.State+".json")); !os.IsNotExist(err) {
		t.Fatalf("pending file should be removed: %v", err)
	}
}
