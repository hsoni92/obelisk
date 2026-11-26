package adminapi

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hsoni92/obelisk/internal/release"
)

func writeMeta(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".install-meta.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckUpdate(t *testing.T) {
	orig := resolveLatest
	defer func() { resolveLatest = orig }()

	t.Run("update available", func(t *testing.T) {
		dir := t.TempDir()
		writeMeta(t, dir, `{"schema":2,"build":5,"source":"release"}`)
		resolveLatest = func(string) (int, error) { return 7, nil }
		st := (&Server{HelmRoot: dir}).checkUpdate(false)
		if !st.UpdateAvailable || st.InstalledBuild != 5 || st.LatestBuild != 7 || st.Error != "" {
			t.Fatalf("got %+v", st)
		}
	})
	t.Run("up to date", func(t *testing.T) {
		dir := t.TempDir()
		writeMeta(t, dir, `{"build":7,"source":"release"}`)
		resolveLatest = func(string) (int, error) { return 7, nil }
		if st := (&Server{HelmRoot: dir}).checkUpdate(false); st.UpdateAvailable {
			t.Fatalf("got %+v", st)
		}
	})
	t.Run("dev install is not nagged", func(t *testing.T) {
		dir := t.TempDir()
		writeMeta(t, dir, `{"build":0,"source":"dev"}`)
		resolveLatest = func(string) (int, error) { return 7, nil }
		if st := (&Server{HelmRoot: dir}).checkUpdate(false); st.UpdateAvailable {
			t.Fatalf("got %+v", st)
		}
	})
	t.Run("no release yet", func(t *testing.T) {
		dir := t.TempDir()
		writeMeta(t, dir, `{"build":1}`)
		resolveLatest = func(string) (int, error) { return 0, release.ErrNoRelease }
		if st := (&Server{HelmRoot: dir}).checkUpdate(false); st.UpdateAvailable || st.Error != "" {
			t.Fatalf("got %+v", st)
		}
	})
	t.Run("missing meta", func(t *testing.T) {
		if st := (&Server{HelmRoot: t.TempDir()}).checkUpdate(false); st.Checkable || st.Error == "" {
			t.Fatalf("got %+v", st)
		}
	})
	t.Run("transient failure keeps last good answer, cache honoured", func(t *testing.T) {
		dir := t.TempDir()
		writeMeta(t, dir, `{"build":5,"source":"release"}`)
		calls := 0
		resolveLatest = func(string) (int, error) { calls++; return 7, nil }
		s := &Server{HelmRoot: dir}
		s.checkUpdate(false)
		s.checkUpdate(false)
		if calls != 1 {
			t.Fatalf("cache bypassed: %d calls", calls)
		}
		resolveLatest = func(string) (int, error) { return 0, errors.New("offline") }
		st := s.checkUpdate(true)
		if !st.UpdateAvailable || st.Error == "" {
			t.Fatalf("got %+v", st)
		}
	})
}
