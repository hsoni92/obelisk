package mail

import "testing"

func TestLookupSMTPKnownDomains(t *testing.T) {
	for _, addr := range []string{"u@gmail.com", "u@outlook.com", "u@yahoo.com"} {
		if _, err := LookupSMTP(addr); err != nil {
			t.Fatalf("%s: %v", addr, err)
		}
	}
	if _, err := LookupSMTP("u@unknown.example"); err == nil {
		t.Fatal("expected unsupported domain")
	}
}
