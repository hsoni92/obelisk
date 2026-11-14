package openrouter

import "testing"

func TestFingerprintKeyStable(t *testing.T) {
	key := "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"
	a := FingerprintKey(key)
	b := FingerprintKey(key)
	if a != b || len(a) != 64 {
		t.Fatalf("fingerprint len=%d a=%s", len(a), a)
	}
	if FingerprintKey(key+" ") != a {
		t.Fatal("trim should match")
	}
}

func TestKeyTail(t *testing.T) {
	key := "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"
	if got := KeyTail(key); got != "b0d32e1f" {
		t.Fatalf("tail=%q want b0d32e1f", got)
	}
}