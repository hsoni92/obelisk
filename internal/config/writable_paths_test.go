package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteConfigPath_ApplianceEtc(t *testing.T) {
	t.Setenv("OBELISK_CONFIG_WRITABLE", "")
	got := WriteConfigPath(DefaultPath)
	if got != WritableApplianceConfigPath {
		t.Fatalf("got %q", got)
	}
}

func TestActiveConfigPath_PrefersWritableCopy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBELISK_CONFIG_WRITABLE", filepath.Join(dir, "live.yaml"))
	live := WriteConfigPath(DefaultPath)
	seed := filepath.Join(dir, "seed.yaml")
	if err := os.WriteFile(seed, []byte("machine:\n  id: obelisk-aaa\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("machine:\n  id: obelisk-bbb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OBELISK_CONFIG_WRITABLE", live)
	got := ActiveConfigPath(seed)
	if got != live {
		t.Fatalf("got %q want %q", got, live)
	}
}

func TestSave_WritesApplianceWritablePath(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib")
	t.Setenv("OBELISK_CONFIG_WRITABLE", filepath.Join(lib, "obelisk.yaml"))
	etc := filepath.Join(dir, "etc", "obelisk.yaml")
	spec, err := Load(etc)
	if err != nil {
		t.Fatal(err)
	}
	spec.Machine.ID = "obelisk-testmachine1"
	if err := SaveQuiet(etc, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(lib, "obelisk.yaml")); err != nil {
		t.Fatal(err)
	}
}
