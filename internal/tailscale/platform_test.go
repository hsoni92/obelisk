package tailscale

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOSReleaseFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "os-release")
	content := `NAME="Ubuntu"
VERSION="24.04 LTS"
ID=ubuntu
ID_LIKE=debian
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := parseOSReleaseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "ubuntu" || p.IDLike != "debian" {
		t.Fatalf("got %+v", p)
	}
}

func TestDistroLabel(t *testing.T) {
	if distroLabel(Platform{ID: "fedora"}) != "fedora" {
		t.Fatal("expected fedora")
	}
	if distroLabel(Platform{OS: "linux"}) != "linux" {
		t.Fatal("expected linux fallback")
	}
}
