package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidToken(t *testing.T) {
	if !ValidToken("abc", "abc") {
		t.Fatal("expected match")
	}
	if ValidToken("abc", "abd") {
		t.Fatal("expected mismatch")
	}
}

func TestWriteReadTokenFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	_ = os.WriteFile(cfg, []byte("apiVersion: obelisk.dev/v1\n"), 0o644)
	if err := WriteToken(cfg, "prj_test", "obd_secret"); err != nil {
		t.Fatal(err)
	}
	got := ReadToken(cfg, "prj_test")
	if got != "obd_secret" {
		t.Fatalf("got %q", got)
	}
	DeleteToken(cfg, "prj_test")
	if ReadToken(cfg, "prj_test") != "" {
		t.Fatal("expected cleared")
	}
}
