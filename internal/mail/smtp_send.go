package mail

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SendMessage sends a simple text/html message via consumer SMTP.
func SendMessage(address, secret, to, subject, text, html string) error {
	prof, err := LookupSMTP(address)
	if err != nil {
		return err
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient required")
	}
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("recipient and subject must not contain line breaks")
	}
	msg := buildMIME(address, to, subject, text, html)
	addr := net.JoinHostPort(prof.Host, strconv.Itoa(prof.Port))
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
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(address); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func buildMIME(from, to, subject, text, html string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	if html != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(html)
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(text)
	}
	return b.String()
}
