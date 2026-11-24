package mail

import (
	"fmt"
	"strings"
)

type SMTPProfile struct {
	Host        string
	Port        int
	ImplicitTLS bool
}

var smtpByDomain = map[string]SMTPProfile{
	// Submission port 587 + STARTTLS: many VPS hosts (e.g. Hetzner) block outbound 465/25 but allow 587.
	"gmail.com":      {Host: "smtp.gmail.com", Port: 587, ImplicitTLS: false},
	"googlemail.com": {Host: "smtp.gmail.com", Port: 587, ImplicitTLS: false},
	"outlook.com":    {Host: "smtp-mail.outlook.com", Port: 587, ImplicitTLS: false},
	"hotmail.com":    {Host: "smtp-mail.outlook.com", Port: 587, ImplicitTLS: false},
	"live.com":       {Host: "smtp-mail.outlook.com", Port: 587, ImplicitTLS: false},
	"yahoo.com":      {Host: "smtp.mail.yahoo.com", Port: 587, ImplicitTLS: false},
}

func LookupSMTP(address string) (SMTPProfile, error) {
	domain := strings.ToLower(strings.TrimSpace(strings.Split(address, "@")[1]))
	if domain == "" {
		return SMTPProfile{}, fmt.Errorf("mailbox address has no domain")
	}
	p, ok := smtpByDomain[domain]
	if !ok {
		return SMTPProfile{}, fmt.Errorf("no SMTP preset for domain %s", domain)
	}
	return p, nil
}
