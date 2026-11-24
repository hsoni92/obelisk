package obelisk

import "testing"

func TestDeriveMachineIDFromAnchorDeterministic(t *testing.T) {
	a := DeriveMachineIDFromAnchor("dmi:550e8400-e29b-41d4-a716-446655440000")
	b := DeriveMachineIDFromAnchor("dmi:550e8400-e29b-41d4-a716-446655440000")
	if a != b {
		t.Fatalf("derive not stable: %q vs %q", a, b)
	}
	if err := ValidateMachineID(a); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveMachineIDFromAnchorDiffersByAnchor(t *testing.T) {
	a := DeriveMachineIDFromAnchor("dmi:aaaa")
	b := DeriveMachineIDFromAnchor("dmi:bbbb")
	if a == b {
		t.Fatal("expected different anchors to yield different ids")
	}
}
