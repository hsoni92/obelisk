package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGitHubOAuthFile(t *testing.T) {
	dir := t.TempDir()
	o := GitHubOAuth{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURI:  "http://example/login/callback",
	}
	path, err := WriteGitHubOAuthFile(dir, o)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", st.Mode().Perm())
	}
	got, ok, err := ReadGitHubOAuthFile(dir)
	if err != nil || !ok {
		t.Fatalf("read: ok=%v err=%v", ok, err)
	}
	if got.ClientID != o.ClientID || got.RedirectURI != o.RedirectURI {
		t.Fatalf("got %#v", got)
	}
}

func TestLoginCallbackURI(t *testing.T) {
	if got := LoginCallbackURI("http://10.0.0.1/"); got != "http://10.0.0.1/login/callback" {
		t.Fatal(got)
	}
	if got := LoginCallbackURI("http://10.0.0.1"); got != "http://10.0.0.1/login/callback" {
		t.Fatal(got)
	}
}

func TestWriteGitHubOAuthFileRequiresFields(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "creds")
	_, err := WriteGitHubOAuthFile(dir, GitHubOAuth{})
	if err == nil {
		t.Fatal("expected error")
	}
}
