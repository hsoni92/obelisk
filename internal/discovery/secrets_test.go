package discovery

import "testing"

func TestValidToken(t *testing.T) {
	if !ValidToken("abc", "abc") {
		t.Fatal("expected match")
	}
	if ValidToken("abc", "abd") {
		t.Fatal("expected mismatch")
	}
	if ValidToken("", "x") || ValidToken("x", "") {
		t.Fatal("empty should fail")
	}
}