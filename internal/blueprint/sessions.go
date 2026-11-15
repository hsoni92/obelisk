package blueprint

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
)

type adminSession struct {
	Login   string    `json:"login"`
	Expires time.Time `json:"expires"`
}

// AdminSessions is the server-side map behind the admin UI cookie.
type AdminSessions struct {
	path string
	mu   sync.Mutex
	all  map[string]adminSession
}

func NewAdminSessions(configPath string) *AdminSessions {
	s := &AdminSessions{
		path: filepath.Join(config.CredentialsDir(configPath), "obelisk-admin-sessions.json"),
		all:  map[string]adminSession{},
	}
	if data, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(data, &s.all)
	}
	if s.all == nil {
		s.all = map[string]adminSession{}
	}
	return s
}

// Create stores a session for the GitHub login and returns the cookie value.
func (s *AdminSessions) Create(login string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.all[id] = adminSession{Login: login, Expires: time.Now().Add(7 * 24 * time.Hour)}
	return id, s.persistLocked()
}

// Lookup returns the GitHub login for a cookie value.
func (s *AdminSessions) Lookup(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.all[id]
	if !ok || time.Now().After(sess.Expires) {
		delete(s.all, id)
		return "", false
	}
	return sess.Login, true
}

// Delete removes a session (sign out).
func (s *AdminSessions) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.all, id)
	_ = s.persistLocked()
}

func (s *AdminSessions) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(s.all)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
