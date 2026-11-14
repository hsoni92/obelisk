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
	"gmail.com":     {Host: "smtp.gmail.com", Port: 465, ImplicitTLS: true},
	"googlemail.com": {Host: "smtp.gmail.com", Port: 465, ImplicitTLS: true},
	"outlook.com":   {Host: "smtp-mail.outlook.com", Port: 587, ImplicitTLS: false},
	"hotmail.com":   {Host: "smtp-mail.outlook.com", Port: 587, ImplicitTLS: false},
	"live.com":      {Host: "smtp-mail.outlook.com", Port: 587, ImplicitTLS: false},
	"yahoo.com":     {Host: "smtp.mail.yahoo.com", Port: 465, ImplicitTLS: true},
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