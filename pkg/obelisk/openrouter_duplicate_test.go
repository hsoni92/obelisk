package obelisk

import "testing"

func TestFindAccountByFingerprint(t *testing.T) {
	accounts := []OpenRouterAccount{
		{ID: "a", KeyFingerprint: "abc"},
		{ID: "b", KeyFingerprint: "def"},
	}
	if _, ok := FindAccountByFingerprint(accounts, "abc", ""); !ok {
		t.Fatal("expected match")
	}
	if _, ok := FindAccountByFingerprint(accounts, "abc", "a"); ok {
		t.Fatal("exclude self")
	}
}