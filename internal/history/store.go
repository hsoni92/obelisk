package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one observed state transition for a component key.
type Event struct {
	At        time.Time `json:"at"`
	Key       string    `json:"key"`
	Kind      string    `json:"kind"` // platform | instance
	State     string    `json:"state"`
	Detail    string    `json:"detail,omitempty"`
	HelmStatus string   `json:"helm_status,omitempty"`
}

const defaultRetention = 24 * time.Hour

// Store appends transition events and prunes by retention.
type Store struct {
	path       string
	retention  time.Duration
	mu         sync.Mutex
	lastState  map[string]string
}

func NewStore(path string) *Store {
	return &Store{
		path:      path,
		retention: defaultRetention,
		lastState: map[string]string{},
	}
}

// Record appends an event when state changed from the last recorded value.
func (s *Store) Record(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.lastState[e.Key]
	if prev == e.State {
		return nil
	}
	s.lastState[e.Key] = e.State
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if err := s.ensureDir(); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(e); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return s.pruneLocked()
}

// Since returns events with At >= since (reads full file; prunes stale lines).
func (s *Store) Since(since time.Time) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDir(); err != nil {
		return nil, err
	}
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		if !e.At.Before(since) {
			out = append(out, e)
		}
		s.lastState[e.Key] = e.State
	}
	return out, sc.Err()
}

func (s *Store) ensureDir() error {
	dir := filepath.Dir(s.path)
	return os.MkdirAll(dir, 0o755)
}

func (s *Store) pruneLocked() error {
	cutoff := time.Now().UTC().Add(-s.retention)
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var keep []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		if !e.At.Before(cutoff) {
			keep = append(keep, e)
		}
	}
	_ = f.Close()
	if len(keep) == 0 {
		return os.Remove(s.path)
	}
	tmp := s.path + ".tmp"
	wf, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(wf)
	for _, e := range keep {
		if err := enc.Encode(e); err != nil {
			_ = wf.Close()
			return err
		}
	}
	if err := wf.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}