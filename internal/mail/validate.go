package mail

import (
	"fmt"
	"strings"
)

// ValidateSMTPAuth checks credentials with a minimal SMTP session (no message sent).
func ValidateSMTPAuth(address, secret string) error {
	address = strings.TrimSpace(address)
	secret = strings.TrimSpace(secret)
	if address == "" || secret == "" {
		return fmt.Errorf("address and secret required")
	}
	if _, err := LookupSMTP(address); err != nil {
		return err
	}
	return smtpAuth(address, secret)
}
