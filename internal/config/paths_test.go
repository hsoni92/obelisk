package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath_PrefersEnv(t *testing.T) {
	t.Setenv("OBELISK_CONFIG", "/tmp/custom/obelisk.yaml")
	got := ResolvePath()
	if got != "/tmp/custom/obelisk.yaml" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureReadablePath_FallsBackFromEtc(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "deploy", "helm", "platform"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := EnsureReadablePath(DefaultPath)
	want := filepath.Join(dir, "dev", "obelisk.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLoad_AutoInitDevPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	spec, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Deployment.Profile != "local" {
		t.Fatalf("profile %q", spec.Deployment.Profile)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("file not created")
	}
}