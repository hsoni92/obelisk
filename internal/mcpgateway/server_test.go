package mcpgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/mailgateway"
)

type fakeMailer struct {
	enabled bool
	sent    []mailgateway.SendRequest
	err     error
}

func (f *fakeMailer) Send(_ context.Context, r mailgateway.SendRequest) (mailgateway.SendResult, error) {
	if f.err != nil {
		return mailgateway.SendResult{}, f.err
	}
	for _, p := range f.sent {
		if p.IdempotencyKey == r.IdempotencyKey {
			return mailgateway.SendResult{AccountID: "a", From: "me@x.com", Deduped: true}, nil
		}
	}
	f.sent = append(f.sent, r)
	return mailgateway.SendResult{AccountID: "a", From: "me@x.com"}, nil
}
func (f *fakeMailer) Status() mailgateway.Status { return mailgateway.Status{Enabled: f.enabled} }

func newTestServer(t *testing.T, m *fakeMailer) *Server {
	t.Helper()
	t.Setenv("OBELISK_CREDENTIALS_DIR", t.TempDir())
	s := NewServer(filepath.Join(t.TempDir(), "obelisk.yaml"), m)
	s.enabled = true
	s.tokens = []string{"obk_mcp_test"}
	return s
}

func rpc(t *testing.T, s *Server, token, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func callBody(name, args string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `}}`
}

func resultText(t *testing.T, out map[string]any) (string, bool) {
	t.Helper()
	res, ok := out["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", out)
	}
	c := res["content"].([]any)[0].(map[string]any)
	return c["text"].(string), res["isError"].(bool)
}

func TestAuth(t *testing.T) {
	s := newTestServer(t, &fakeMailer{})
	rec, _ := rpc(t, s, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	if rec.Code != 401 || rec.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	rec, _ = rpc(t, s, "wrong", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	if rec.Code != 401 {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	s.enabled = false
	rec, _ = rpc(t, s, "obk_mcp_test", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	if rec.Code != 401 {
		t.Fatalf("disabled must be 401, got %d", rec.Code)
	}
	if s.Status().AuthFailures != 3 {
		t.Fatalf("auth failures = %d", s.Status().AuthFailures)
	}
}

func TestProtocol(t *testing.T) {
	s := newTestServer(t, &fakeMailer{})
	_, out := rpc(t, s, "obk_mcp_test", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if out["result"].(map[string]any)["protocolVersion"] != "2025-06-18" {
		t.Fatal("version not negotiated")
	}
	rec, _ := rpc(t, s, "obk_mcp_test", `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if rec.Code != 202 {
		t.Fatalf("notification code %d", rec.Code)
	}
	_, out = rpc(t, s, "obk_mcp_test", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`)
	if out["error"].(map[string]any)["code"].(float64) != -32600 {
		t.Fatal("batch must be rejected")
	}
	_, out = rpc(t, s, "obk_mcp_test", `{"jsonrpc":"2.0","id":1,"method":"nope"}`)
	if out["error"].(map[string]any)["code"].(float64) != -32601 {
		t.Fatal("unknown method")
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer obk_mcp_test")
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("GET should be 405, got %d", rec.Code)
	}
}

func TestToolsAlwaysListed(t *testing.T) {
	s := newTestServer(t, &fakeMailer{enabled: false})
	_, out := rpc(t, s, "obk_mcp_test", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := out["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("want both tools listed, got %d", len(tools))
	}
	// No SearXNG and no mail: both explain what is missing.
	_, out = rpc(t, s, "obk_mcp_test", callBody("web_search", `{"query":"x"}`))
	if txt, isErr := resultText(t, out); !isErr || !strings.Contains(txt, "SearXNG") {
		t.Fatalf("search: %q %v", txt, isErr)
	}
	s.toggles = map[string]bool{"send_mail": true}
	_, out = rpc(t, s, "obk_mcp_test", callBody("send_mail", `{"to":"a@b.com","subject":"s","text":"t"}`))
	if txt, isErr := resultText(t, out); !isErr || !strings.Contains(txt, "mail gateway") {
		t.Fatalf("mail: %q %v", txt, isErr)
	}
}

func TestSendMailOffByDefault(t *testing.T) {
	m := &fakeMailer{enabled: true}
	s := newTestServer(t, m)
	_, out := rpc(t, s, "obk_mcp_test", callBody("send_mail", `{"to":"a@b.com","subject":"s","text":"t"}`))
	if txt, isErr := resultText(t, out); !isErr || !strings.Contains(txt, "switched off") {
		t.Fatalf("%q %v", txt, isErr)
	}
	if len(m.sent) != 0 {
		t.Fatal("sent while disabled")
	}
}

func TestSendMailValidationLimitsAndLog(t *testing.T) {
	m := &fakeMailer{enabled: true}
	s := newTestServer(t, m)
	s.toggles = map[string]bool{"send_mail": true}
	send := func(args string) (string, bool) {
		_, out := rpc(t, s, "obk_mcp_test", callBody("send_mail", args))
		return resultText(t, out)
	}
	for _, bad := range []string{
		`{"to":"a@b.com, c@d.com","subject":"s","text":"t"}`,
		`{"to":"a@b.com","subject":"s\nBcc: x@y.com","text":"t"}`,
		`{"to":"a@b.com","subject":"","text":"t"}`,
		`{"to":"Name <a@b.com>","subject":"s","text":"t"}`,
	} {
		if _, isErr := send(bad); !isErr {
			t.Fatalf("accepted %s", bad)
		}
	}
	if txt, isErr := send(`{"to":"a@b.com","subject":"hi","text":"body"}`); isErr || !strings.Contains(txt, "a@b.com") {
		t.Fatalf("%q %v", txt, isErr)
	}
	if txt, isErr := send(`{"to":"a@b.com","subject":"hi","text":"body"}`); isErr || !strings.Contains(txt, "identical") {
		t.Fatalf("dedupe: %q %v", txt, isErr)
	}
	if len(m.sent) != 1 {
		t.Fatalf("sent %d", len(m.sent))
	}
	for i := 1; i < maxMailPerHour; i++ {
		send(`{"to":"a@b.com","subject":"n` + string(rune('a'+i)) + `","text":"body"}`)
	}
	if txt, isErr := send(`{"to":"a@b.com","subject":"over","text":"body"}`); !isErr || !strings.Contains(txt, "Hourly") {
		t.Fatalf("limit: %q %v", txt, isErr)
	}
	log := s.SendLog(50)
	if len(log) != maxMailPerHour+1 || log[0].Outcome != "ok" && log[0].Outcome != "deduped" {
		t.Fatalf("log len %d", len(log))
	}
	// The log (and so the cap) survives a restart.
	s2 := NewServer(s.ConfigPath, m)
	if s2.sendlog.sent(time.Now().Add(-time.Hour)) != maxMailPerHour {
		t.Fatal("cap lost on restart")
	}
}

func TestWebSearch(t *testing.T) {
	searx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			w.WriteHeader(403)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"A​ title","url":"https://a.example","content":"` + strings.Repeat("x", 600) + `"},{"title":"B","url":"https://b.example","content":"b"}]}`))
	}))
	defer searx.Close()
	s := newTestServer(t, &fakeMailer{})
	s.searxURL = searx.URL
	_, out := rpc(t, s, "obk_mcp_test", callBody("web_search", `{"query":"go","limit":1}`))
	txt, isErr := resultText(t, out)
	if isErr || !strings.Contains(txt, "https://a.example") || strings.Contains(txt, "b.example") {
		t.Fatalf("%q %v", txt, isErr)
	}
	if strings.Contains(txt, "​") || len(txt) > searchMaxBytes+200 {
		t.Fatal("not cleaned or capped")
	}
	s.searxURL = "http://127.0.0.1:1"
	_, out = rpc(t, s, "obk_mcp_test", callBody("web_search", `{"query":"go"}`))
	if txt, isErr := resultText(t, out); !isErr || !strings.Contains(txt, "not reachable") {
		t.Fatalf("%q", txt)
	}
}

func TestRotateToken(t *testing.T) {
	t.Setenv("OBELISK_CREDENTIALS_DIR", t.TempDir())
	cfg := filepath.Join(t.TempDir(), "obelisk.yaml")
	tok, err := RotateToken(cfg)
	if err != nil || !strings.HasPrefix(tok, TokenPrefix) {
		t.Fatal(tok, err)
	}
	c, ok, _ := LoadCredentials(cfg)
	if !ok || c.Tokens[0].Value != tok {
		t.Fatal("not stored")
	}
}
