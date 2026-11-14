package obelisk

import "testing"

func TestFindMailAccountByAddress(t *testing.T) {
	accounts := []MailAccount{
		{ID: "m1", Address: "a@gmail.com"},
		{ID: "m2", Address: "b@yahoo.com"},
	}
	if dup, ok := FindMailAccountByAddress(accounts, "A@Gmail.com", ""); !ok || dup.ID != "m1" {
		t.Fatalf("expected duplicate m1, got %v %v", dup, ok)
	}
}

func TestValidateMailEnabledNeedsAccounts(t *testing.T) {
	if err := ValidateMail(MailGateway{Enabled: true}); err == nil {
		t.Fatal("expected error")
	}
}