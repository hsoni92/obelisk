package openrouter

import "testing"

func TestFingerprintKeyStable(t *testing.T) {
	key := "sk-or-v1-000000000000000000000000000000000000000000000000000deadbeef"
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
	key := "sk-or-v1-000000000000000000000000000000000000000000000000000deadbeef"
	if got := KeyTail(key); got != "deadbeef" {
		t.Fatalf("tail=%q want deadbeef", got)
	}
}
