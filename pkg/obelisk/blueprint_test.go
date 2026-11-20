package obelisk

import "testing"

func TestBlueprintUpstreamBranch(t *testing.T) {
	id := "obelisk-abc123def456"
	got, err := BlueprintUpstreamBranch(id)
	if err != nil {
		t.Fatal(err)
	}
	want := BlueprintBranchPrefix + id
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEffectiveBlueprintBranchDefault(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Machine.ID = "obelisk-abc123def456"
	spec.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
	got := EffectiveBlueprintBranch(spec)
	want := "machine-obelisk-abc123def456"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEffectiveBlueprintBranchReplacesLegacyMain(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Machine.ID = "obelisk-abc123def456"
	spec.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
	spec.Blueprint.GitHub.Branch = "main"
	got := EffectiveBlueprintBranch(spec)
	want := "machine-obelisk-abc123def456"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
