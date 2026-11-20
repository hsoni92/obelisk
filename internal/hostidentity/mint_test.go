package hostidentity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestMintUsesPersistedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appliance-id")
	const want = "obelisk-aaaaaaaaaaaa"
	if err := os.WriteFile(path, []byte(want+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OBELISK_APPLIANCE_ID_FILE", path)
	t.Setenv("OBELISK_MACHINE_ANCHOR", "")

	got := Mint()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMintDerivesFromAnchor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBELISK_APPLIANCE_ID_FILE", filepath.Join(dir, "appliance-id"))
	t.Setenv("OBELISK_MACHINE_ANCHOR", "dmi:test-anchor-123")

	got := Mint()
	want := obelisk.DeriveMachineIDFromAnchor("dmi:test-anchor-123")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	data, err := os.ReadFile(ApplianceIDPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != got+"\n" {
		t.Fatalf("persist mismatch: %q", data)
	}
}

func TestValidDMIUUID(t *testing.T) {
	if validDMIUUID("ffffffff-ffff-ffff-ffff-ffffffffffff") {
		t.Fatal("sentinel uuid should be rejected")
	}
	if !validDMIUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("expected valid uuid")
	}
}