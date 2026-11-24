package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath_PrefersApplianceLiveCopy(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "obelisk.yaml")
	if err := os.WriteFile(live, []byte("deployment:\n  profile: vps\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OBELISK_CONFIG", "")
	t.Setenv("OBELISK_CONFIG_WRITABLE", live)
	got := ResolvePath()
	if got != live {
		t.Fatalf("got %q want %q", got, live)
	}
}

func TestResolvePath_PrefersEnv(t *testing.T) {
	t.Setenv("OBELISK_CONFIG", "/tmp/custom/obelisk.yaml")
	got := ResolvePath()
	if got != "/tmp/custom/obelisk.yaml" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureReadablePath_ApplianceLiveWithoutEtcSeed(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "var", "lib", "obelisk", "obelisk.yaml")
	if err := os.MkdirAll(filepath.Dir(live), 0o700); err != nil {
		t.Fatal(err)
	}
	const body = "deployment:\n  profile: vps\nmachine:\n  id: obelisk-aaaaaaa1\n"
	if err := os.WriteFile(live, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OBELISK_CONFIG_WRITABLE", live)
	got := EnsureReadablePath(DefaultPath)
	if got != live {
		t.Fatalf("got %q want live copy %q", got, live)
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
