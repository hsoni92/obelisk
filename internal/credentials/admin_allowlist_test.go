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