package blueprint

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/credentials"
)

func allowlistTestRuntime(t *testing.T) (*Runtime, string) {
	t.Helper()
	rt := newStatusRuntime(t, "", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	return rt, config.CredentialsDir(rt.ConfigPath)
}

// The login page reads this flag to tell the operator that nobody is allowed yet.
func TestStatus_loginAllowlistEmptyOnFreshInstall(t *testing.T) {
	rt, credDir := allowlistTestRuntime(t)
	if _, err := credentials.InitAdminGitHubAllowlist(credDir); err != nil {
		t.Fatal(err)
	}
	st := rt.Status(context.Background(), "")
	if !st.LoginAllowlistEmpty {
		t.Fatal("fresh install (empty allowlist) must report login_allowlist_empty")
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"login_allowlist_empty":true`) {
		t.Fatalf("status JSON must carry login_allowlist_empty: %s", b)
	}
}

func TestStatus_loginAllowlistEmptyClearsWhenUserAdded(t *testing.T) {
	rt, credDir := allowlistTestRuntime(t)
	if _, err := credentials.InitAdminGitHubAllowlist(credDir); err != nil {
		t.Fatal(err)
	}
	if _, err := credentials.AddAdminGitHubLogin(credDir, "alice"); err != nil {
		t.Fatal(err)
	}
	if rt.Status(context.Background(), "").LoginAllowlistEmpty {
		t.Fatal("allowlist with a user must not report empty")
	}
}

func TestStatus_loginAllowlistEmptyFalseForLegacyOpenInstall(t *testing.T) {
	rt, credDir := allowlistTestRuntime(t)
	if filepath.Base(credDir) == "" {
		t.Fatal("no credentials dir")
	}
	if rt.Status(context.Background(), "").LoginAllowlistEmpty {
		t.Fatal("no allowlist file is the legacy open state, not 'no users allowed'")
	}
}
