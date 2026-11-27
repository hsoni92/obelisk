package mailgateway

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/hsoni92/obelisk/internal/mail"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// SendRequest is one outbound message. Only a single recipient is supported.
type SendRequest struct {
	To             string
	Subject        string
	Text           string
	HTML           string
	AccountID      string
	Failover       bool
	IdempotencyKey string
}

// SendResult describes an accepted message.
type SendResult struct {
	AccountID string
	From      string
	Deduped   bool
}

// SendError carries an HTTP status and a message safe to show to callers.
type SendError struct {
	Status int
	Msg    string
}

func (e *SendError) Error() string { return e.Msg }

func sendErr(status int, msg string) *SendError { return &SendError{Status: status, Msg: msg} }

// Send picks a mailbox from the pool (or the pinned account) and sends one message.
// It does not check the gateway token; callers authenticate before calling it.
func (s *Server) Send(_ context.Context, req SendRequest) (SendResult, error) {
	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey != "" {
		if prev, ok := s.idem.get(idemKey); ok {
			return SendResult{AccountID: prev.accountID, From: prev.from, Deduped: true}, nil
		}
	}
	to := strings.TrimSpace(req.To)
	if to == "" || strings.TrimSpace(req.Subject) == "" {
		return SendResult{}, sendErr(http.StatusBadRequest, "to and subject required")
	}
	text := strings.TrimSpace(req.Text)
	html := strings.TrimSpace(req.HTML)
	if text == "" && html == "" {
		return SendResult{}, sendErr(http.StatusBadRequest, "text or html required")
	}

	s.mu.RLock()
	sel := s.selector
	s.mu.RUnlock()
	var candidates []mailAccount
	if id := strings.TrimSpace(req.AccountID); id != "" {
		acc, ok := sel.ByID(id)
		if !ok {
			return SendResult{}, sendErr(http.StatusNotFound, "unknown account_id")
		}
		candidates = []mailAccount{acc}
		if req.Failover {
			for _, a := range sel.RotateList() {
				if a.ID != acc.ID {
					candidates = append(candidates, a)
				}
			}
		}
	} else {
		candidates = sel.RotateList()
	}

	var lastErr error
	for _, acc := range candidates {
		err := mail.SendMessage(acc.Address, acc.Secret, to, req.Subject, text, html)
		if err != nil {
			lastErr = err
			if !retryableSMTP(err) {
				break
			}
			continue
		}
		s.mu.Lock()
		s.lastAccountID = acc.ID
		s.mu.Unlock()
		if idemKey != "" {
			s.idem.put(idemKey, acc.ID, acc.Address)
		}
		log.Printf("mail_send account_id=%s provider=%s outcome=ok recipient_domain=%s", acc.ID, obelisk.MailboxDomain(acc.Address), domainOf(to))
		return SendResult{AccountID: acc.ID, From: acc.Address}, nil
	}
	msg := "could not send mail"
	if lastErr != nil {
		msg = publicError(lastErr)
	}
	return SendResult{}, sendErr(http.StatusBadGateway, msg)
}
