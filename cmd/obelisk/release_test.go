package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseBuildTag(t *testing.T) {
	cases := map[string]struct {
		n  int
		ok bool
	}{
		"build-142": {142, true},
		"build-9":   {9, true},
		"build-10":  {10, true},
		"build-":    {0, false},
		"v1.2.3":    {0, false},
		"main":      {0, false},
		"build-1a":  {0, false},
		"":          {0, false},
	}
	for in, want := range cases {
		n, ok := parseBuildTag(in)
		if n != want.n || ok != want.ok {
			t.Errorf("parseBuildTag(%q) = %d,%v want %d,%v", in, n, ok, want.n, want.ok)
		}
	}
	// numeric, not lexical: build-9 < build-10
	a, _ := parseBuildTag("build-9")
	b, _ := parseBuildTag("build-10")
	if a >= b {
		t.Fatalf("build-9 (%d) must be lower than build-10 (%d)", a, b)
	}
}

func TestResolveLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/o/r/releases/latest":
			http.Redirect(w, r, "/o/r/releases/tag/build-42", http.StatusFound)
		case "/o/r/releases/tag/build-7":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	n, err := resolveRelease(srv.URL, "o/r", "")
	if err != nil || n != 42 {
		t.Fatalf("latest = %d, %v; want 42", n, err)
	}
	n, err = resolveRelease(srv.URL, "o/r", "build-7")
	if err != nil || n != 7 {
		t.Fatalf("pinned = %d, %v; want 7", n, err)
	}
	if _, err = resolveRelease(srv.URL, "o/r", "build-8"); err == nil {
		t.Fatal("missing release must error")
	}
	if _, err = resolveRelease(srv.URL, "o/r", "main"); err == nil {
		t.Fatal("non build-N tag must error")
	}
}

func TestResolveNoReleaseAndServerError(t *testing.T) {
	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	if _, err := resolveRelease(notFound.URL, "o/r", ""); !errors.Is(err, errNoRelease) {
		t.Fatalf("404 on latest must be errNoRelease, got %v", err)
	}

	limited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer limited.Close()
	_, err := resolveRelease(limited.URL, "o/r", "")
	if err == nil || errors.Is(err, errNoRelease) {
		t.Fatalf("403 must be an error that is not 'no release', got %v", err)
	}

	badTag := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/o/r/releases/tag/v1.0.0", http.StatusFound)
	}))
	defer badTag.Close()
	if _, err := resolveRelease(badTag.URL, "o/r", ""); err == nil {
		t.Fatal("latest release with non build-N tag must error")
	}
}

func TestReadInstallMetaLegacyIsBuildZero(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"repo":"hsoni92/obelisk","ref":"main","installed_at":"2026-11-01T00:00:00Z","image_tag":"0.1.0-main-20261101T000000Z"}`
	if err := os.WriteFile(filepath.Join(dir, ".install-meta.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _, err := readInstallMeta(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Build != 0 || m.Source != "" {
		t.Fatalf("legacy meta must read as build 0 with no source, got %+v", m)
	}

	v2 := `{"schema":2,"repo":"o/r","ref":"build-5","tag":"build-5","build":5,"source":"release","arch":"arm64","profile":"local","image_tag":"build-5","installed_at":"x"}`
	if err := os.WriteFile(filepath.Join(dir, ".install-meta.json"), []byte(v2), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _, err = readInstallMeta(dir)
	if err != nil || m.Build != 5 || m.Arch != "arm64" || m.Profile != "local" || m.Source != "release" {
		t.Fatalf("v2 meta parsed wrong: %+v, %v", m, err)
	}
}

func TestReleaseRepoPrecedence(t *testing.T) {
	t.Setenv("OBELISK_REPO", "")
	if got := releaseRepo(""); got != defaultRepo {
		t.Fatalf("default = %q", got)
	}
	if got := releaseRepo("fork/obelisk"); got != "fork/obelisk" {
		t.Fatalf("meta repo = %q", got)
	}
	t.Setenv("OBELISK_REPO", "env/obelisk")
	if got := releaseRepo("fork/obelisk"); got != "env/obelisk" {
		t.Fatalf("env repo = %q", got)
	}
}
