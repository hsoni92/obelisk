package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStore_RecordAndSince(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "h.jsonl"))
	if err := s.Record(Event{Key: "redis/a", Kind: "instance", State: "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(Event{Key: "redis/a", Kind: "instance", State: "running"}); err != nil {
		t.Fatal(err)
	}
	// duplicate state — no third line
	if err := s.Record(Event{Key: "redis/a", Kind: "instance", State: "running"}); err != nil {
		t.Fatal(err)
	}
	events, err := s.Since(time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "h.jsonl"))
	if count := len(splitLines(string(raw))); count != 2 {
		t.Fatalf("want 2 lines on disk, got %d", count)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	return out
}