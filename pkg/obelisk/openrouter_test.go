package obelisk

import "testing"

func TestNormalizeOpenRouterDedupesIDs(t *testing.T) {
	spec := Spec{
		OpenRouterGateway: OpenRouterGateway{
			Accounts: []OpenRouterAccount{
				{ID: "a", Label: "one"},
				{ID: "a", Label: "dup"},
				{ID: "", Label: "skip"},
			},
		},
	}
	NormalizeOpenRouter(&spec)
	if len(spec.OpenRouterGateway.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(spec.OpenRouterGateway.Accounts))
	}
	if spec.OpenRouterGateway.Accounts[0].Label != "one" {
		t.Fatalf("unexpected label %q", spec.OpenRouterGateway.Accounts[0].Label)
	}
}
