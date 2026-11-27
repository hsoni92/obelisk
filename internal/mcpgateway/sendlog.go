package mcpgateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
)

const sendLogMax = 200

// SendLogEntry is one send_mail attempt. Bodies are never stored, only a short hash.
type SendLogEntry struct {
	Time      time.Time `json:"time"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Bytes     int       `json:"bytes"`
	BodyHash  string    `json:"body_hash"`
	Outcome   string    `json:"outcome"` // ok | deduped | error
	AccountID string    `json:"account_id,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// sendLog is a bounded, persisted record of send_mail calls. The same data
// drives the per-hour and per-day caps, so the caps survive a restart.
type sendLog struct {
	mu      sync.Mutex
	path    string
	entries []SendLogEntry
	now     func() time.Time
}

func newSendLog(configPath string) *sendLog {
	l := &sendLog{
		path: filepath.Join(config.CredentialsDir(configPath), sendLogFile),
		now:  time.Now,
	}
	if data, err := os.ReadFile(l.path); err == nil {
		_ = json.Unmarshal(data, &l.entries)
	}
	return l
}

func bodyHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:6])
}

func (l *sendLog) add(e SendLogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.Time.IsZero() {
		e.Time = l.now().UTC()
	}
	l.entries = append(l.entries, e)
	if len(l.entries) > sendLogMax {
		l.entries = append([]SendLogEntry(nil), l.entries[len(l.entries)-sendLogMax:]...)
	}
	l.persistLocked()
}

func (l *sendLog) persistLocked() {
	if l.path == "" {
		return
	}
	data, err := json.Marshal(l.entries)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, l.path)
}

// sent counts accepted (non-deduped) sends since the cutoff.
func (l *sendLog) sent(since time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, e := range l.entries {
		if e.Outcome == "ok" && e.Time.After(since) {
			n++
		}
	}
	return n
}

// recent returns entries newest first.
func (l *sendLog) recent(limit int) []SendLogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]SendLogEntry, 0, limit)
	for i := len(l.entries) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, l.entries[i])
	}
	return out
}
