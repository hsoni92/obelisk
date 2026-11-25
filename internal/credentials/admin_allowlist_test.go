package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoginPermittedOpenWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	if err := LoginPermitted(dir, "anyone"); err != nil {
		t.Fatal(err)
	}
}

func TestLoginPermittedEnforced(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteAdminGitHubAllowlist(dir, AdminGitHubAllowlist{Logins: []string{"alice"}}); err != nil {
		t.Fatal(err)
	}
	if err := LoginPermitted(dir, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := LoginPermitted(dir, "Alice"); err != nil {
		t.Fatal(err)
	}
	if err := LoginPermitted(dir, "bob"); !errors.Is(err, ErrLoginNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestLoginPermittedEmptyListDeniesAll(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteAdminGitHubAllowlist(dir, AdminGitHubAllowlist{}); err != nil {
		t.Fatal(err)
	}
	if err := LoginPermitted(dir, "alice"); !errors.Is(err, ErrLoginNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestAddRemoveAndClearAllowlist(t *testing.T) {
	dir := t.TempDir()
	list, err := AddAdminGitHubLogin(dir, "one")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Logins) != 1 {
		t.Fatalf("list %#v", list)
	}
	list, err = AddAdminGitHubLogin(dir, "two")
	if err != nil || len(list.Logins) != 2 {
		t.Fatalf("list %#v err=%v", list, err)
	}
	list, removed, err := RemoveAdminGitHubLogin(dir, "one")
	if err != nil || !removed || len(list.Logins) != 1 || list.Logins[0] != "two" {
		t.Fatalf("list %#v removed=%v err=%v", list, removed, err)
	}
	if err := RemoveAdminGitHubAllowlist(dir); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := ReadAdminGitHubAllowlist(dir); err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	path := filepath.Join(dir, adminAllowlistFile)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stat: %v", err)
	}
}

func TestWriteAllowlistMode0600(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteAdminGitHubAllowlist(dir, AdminGitHubAllowlist{Logins: []string{"u"}})
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
}

func TestInitAdminAllowlistCreatesEmptyDenyAll(t *testing.T) {
	dir := t.TempDir()
	created, err := InitAdminGitHubAllowlist(dir)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if !NoUsersAllowed(dir) {
		t.Fatal("fresh allowlist must report no users allowed")
	}
	if err := LoginPermitted(dir, "anyone"); !errors.Is(err, ErrLoginNotAllowed) {
		t.Fatalf("fresh install must deny everyone, got %v", err)
	}
	// File must exist on disk and be private.
	st, err := os.Stat(filepath.Join(dir, adminAllowlistFile))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("allowlist file: %v mode=%v", err, st)
	}
}

func TestInitAdminAllowlistKeepsExistingList(t *testing.T) {
	dir := t.TempDir()
	if _, err := AddAdminGitHubLogin(dir, "alice"); err != nil {
		t.Fatal(err)
	}
	created, err := InitAdminGitHubAllowlist(dir)
	if err != nil || created {
		t.Fatalf("existing list must not be recreated: created=%v err=%v", created, err)
	}
	if err := LoginPermitted(dir, "alice"); err != nil {
		t.Fatalf("alice must stay allowed: %v", err)
	}
	if NoUsersAllowed(dir) {
		t.Fatal("list with a user is not 'no users allowed'")
	}
}

func TestNoUsersAllowedStates(t *testing.T) {
	dir := t.TempDir()
	if NoUsersAllowed(dir) {
		t.Fatal("missing file is the legacy open state, not 'no users allowed'")
	}
	if err := os.WriteFile(filepath.Join(dir, adminAllowlistFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if NoUsersAllowed(dir) {
		t.Fatal("unreadable file must not claim 'no users allowed'")
	}
	if _, err := WriteAdminGitHubAllowlist(dir, AdminGitHubAllowlist{Logins: []string{"bob"}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RemoveAdminGitHubLogin(dir, "bob"); err != nil {
		t.Fatal(err)
	}
	if !NoUsersAllowed(dir) {
		t.Fatal("removing the last user leaves an enforced empty list: no users allowed")
	}
}

func TestEmptyAllowlistIsWrittenAsEmptyArray(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitAdminGitHubAllowlist(dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, adminAllowlistFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{\"logins\":[]}\n" {
		t.Fatalf("empty allowlist must be {\"logins\":[]}, got %q", b)
	}
}
