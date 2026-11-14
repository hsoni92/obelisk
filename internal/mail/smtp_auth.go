package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

func smtpAuth(address, secret string) error {
	prof, err := LookupSMTP(address)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", prof.Host, prof.Port)
	var client *smtp.Client
	if prof.ImplicitTLS {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, &tls.Config{ServerName: prof.Host})
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, prof.Host)
		if err != nil {
			_ = conn.Close()
			return err
		}
	} else {
		conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, prof.Host)
		if err != nil {
			_ = conn.Close()
			return err
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: prof.Host}); err != nil {
				_ = client.Close()
				return err
			}
		}
	}
	defer client.Close()
	auth := smtp.PlainAuth("", address, secret, prof.Host)
	return client.Auth(auth)
}