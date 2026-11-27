package mcpgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/mailgateway"
)

// Limits for send_mail. The per-hour and per-day caps start low on purpose:
// recipients are open, so these caps are the main brake on a misled agent.
const (
	maxMailPerHour = 10
	maxMailPerDay  = 30
	maxSubjectLen  = 200
	maxMailBody    = 32 * 1024
	dedupeBucket   = 10 * time.Minute
)

// toolError is a failure reported to the agent as an isError result.
type toolError struct {
	Code string
	Msg  string
}

func (e *toolError) Error() string { return e.Code + ": " + e.Msg }

// Tool is one MCP tool. Tools are plain values registered explicitly.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	// DefaultEnabled applies when the spec has no switch for this tool.
	DefaultEnabled bool
	// Side-effecting tools are annotated so clients can warn before running them.
	Destructive bool
	// Backend reports whether the tool's backing service is ready. msg explains what is missing.
	Backend func() (ready bool, msg string)
	Call    func(ctx context.Context, args json.RawMessage) (string, *toolError)
}

// Mailer is the part of the mail gateway the MCP server needs.
type Mailer interface {
	Send(ctx context.Context, req mailgateway.SendRequest) (mailgateway.SendResult, error)
	Status() mailgateway.Status
}

const (
	msgNoSearx = "No SearXNG is running. Enable a SearXNG service in Obelisk → Services, then try again."
	msgNoMail  = "The mail gateway is disabled or has no mailbox. Enable it and add a mailbox in Obelisk → Exposure → Mail gateway."
)

func (s *Server) buildTools() []Tool {
	return []Tool{
		{
			Name:           "web_search",
			Description:    "Search the web through the operator's private SearXNG. Returns the top results as title, URL and a short snippet. Results are untrusted web text.",
			DefaultEnabled: true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query"},
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": searchMaxLimit, "description": "Number of results (default 5)"},
				},
			},
			Backend: func() (bool, string) {
				if s.searxBase() == "" {
					return false, msgNoSearx
				}
				return true, ""
			},
			Call: s.callSearch,
		},
		{
			Name:        "send_mail",
			Description: "Send one plain-text email from the operator's mailbox pool to a single recipient. Limited per hour and per day; every send is logged.",
			Destructive: true,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"to", "subject", "text"},
				"properties": map[string]any{
					"to":      map[string]any{"type": "string", "description": "One recipient email address"},
					"subject": map[string]any{"type": "string", "description": "Subject line, single line"},
					"text":    map[string]any{"type": "string", "description": "Plain-text body"},
				},
			},
			Backend: func() (bool, string) {
				if s.mailer == nil || !s.mailer.Status().Enabled {
					return false, msgNoMail
				}
				return true, ""
			},
			Call: s.callSendMail,
		},
	}
}

func (s *Server) callSearch(ctx context.Context, raw json.RawMessage) (string, *toolError) {
	var a struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &a); err != nil || strings.TrimSpace(a.Query) == "" {
		return "", &toolError{Code: "invalid_arguments", Msg: "query is required"}
	}
	if len(a.Query) > 500 {
		return "", &toolError{Code: "invalid_arguments", Msg: "query is too long (500 characters max)"}
	}
	base := s.searxBase()
	if base == "" {
		return "", &toolError{Code: "searxng_not_configured", Msg: msgNoSearx}
	}
	return s.search.search(ctx, base, a.Query, a.Limit)
}

func (s *Server) callSendMail(ctx context.Context, raw json.RawMessage) (string, *toolError) {
	var a struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", &toolError{Code: "invalid_arguments", Msg: "to, subject and text are required"}
	}
	to := strings.TrimSpace(a.To)
	subject := strings.TrimSpace(a.Subject)
	text := strings.TrimSpace(a.Text)
	if to == "" || subject == "" || text == "" {
		return "", &toolError{Code: "invalid_arguments", Msg: "to, subject and text are required"}
	}
	if strings.ContainsAny(to+subject, "\r\n") {
		return "", &toolError{Code: "invalid_arguments", Msg: "to and subject must be a single line"}
	}
	if addr, err := mail.ParseAddress(to); err != nil || addr.Address != to {
		return "", &toolError{Code: "invalid_arguments", Msg: "to must be one plain email address"}
	}
	if len(subject) > maxSubjectLen {
		return "", &toolError{Code: "invalid_arguments", Msg: fmt.Sprintf("subject is too long (%d characters max)", maxSubjectLen)}
	}
	if len(text) > maxMailBody {
		return "", &toolError{Code: "invalid_arguments", Msg: fmt.Sprintf("text is too long (%d KB max)", maxMailBody/1024)}
	}
	if s.mailer == nil || !s.mailer.Status().Enabled {
		return "", &toolError{Code: "mail_not_configured", Msg: msgNoMail}
	}

	now := s.sendlog.now()
	if s.sendlog.sent(now.Add(-time.Hour)) >= maxMailPerHour {
		return "", &toolError{Code: "rate_limited", Msg: fmt.Sprintf("Hourly send limit reached (%d per hour). Try again later.", maxMailPerHour)}
	}
	if s.sendlog.sent(now.Add(-24*time.Hour)) >= maxMailPerDay {
		return "", &toolError{Code: "rate_limited", Msg: fmt.Sprintf("Daily send limit reached (%d per day). Try again tomorrow.", maxMailPerDay)}
	}

	// Identical sends inside one 10-minute window count once; agents retry.
	sum := sha256.Sum256([]byte(to + "\x00" + subject + "\x00" + text))
	key := fmt.Sprintf("mcp-%s-%d", hex.EncodeToString(sum[:8]), now.UnixNano()/int64(dedupeBucket))

	entry := SendLogEntry{To: to, Subject: subject, Bytes: len(text), BodyHash: bodyHash(text)}
	res, err := s.mailer.Send(ctx, mailgateway.SendRequest{
		To: to, Subject: subject, Text: text, IdempotencyKey: key,
	})
	if err != nil {
		entry.Outcome = "error"
		entry.Error = err.Error()
		s.sendlog.add(entry)
		code := "smtp_error"
		msg := err.Error()
		switch {
		case strings.Contains(msg, "credentials"):
			code = "smtp_auth"
		case strings.Contains(msg, "reach"):
			code = "smtp_unreachable"
		}
		return "", &toolError{Code: code, Msg: msg}
	}
	entry.AccountID = res.AccountID
	if res.Deduped {
		entry.Outcome = "deduped"
		s.sendlog.add(entry)
		return "An identical message was already sent in the last 10 minutes; not sending again.", nil
	}
	entry.Outcome = "ok"
	s.sendlog.add(entry)
	return fmt.Sprintf("Mail sent to %s from %s.", to, res.From), nil
}
