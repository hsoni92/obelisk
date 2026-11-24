package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvisionedStateDir(t *testing.T) {
	dir := t.TempDir()
	if provisionedStateDir(dir) {
		t.Fatal("empty dir is not provisioned")
	}
	if err := os.MkdirAll(filepath.Join(dir, "credentials"), 0o700); err != nil {
		t.Fatal(err)
	}
	if provisionedStateDir(dir) {
		t.Fatal("empty credentials dir is not provisioned (fresh install creates it)")
	}
	if err := os.WriteFile(filepath.Join(dir, "appliance-id"), []byte("obelisk-x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !provisionedStateDir(dir) {
		t.Fatal("appliance-id marks a provisioned appliance")
	}
}

func TestExistingApplianceStateIgnoresOtherPaths(t *testing.T) {
	if existingApplianceState(filepath.Join(t.TempDir(), "obelisk.yaml")) {
		t.Fatal("non-appliance path must not count as provisioned")
	}
}

func TestLoadMissingDevConfigStillAutoInits(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "obelisk.yaml")); err != nil {
		t.Fatalf("fresh dev path should auto-init: %v", err)
	}
}
