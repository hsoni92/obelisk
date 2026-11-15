package blueprint

import "sync"

// Session holds the passphrase-derived key in process memory until lock or exit.
// There is no TTL. The passphrase is not stored.
type Session struct {
	mu   sync.RWMutex
	key  []byte
	id   string
	kdf  KDFParams
	open bool
}

// Set installs a derived key. The caller may discard the passphrase.
func (s *Session) Set(key []byte, kdf KDFParams) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = append([]byte(nil), key...)
	s.id = KeyID(key)
	s.kdf = cloneKDF(kdf)
	s.open = true
}

// Clear drops the key.
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.key {
		s.key[i] = 0
	}
	s.key = nil
	s.id = ""
	s.kdf = KDFParams{}
	s.open = false
}

// Unlocked reports whether encrypt/push may run.
func (s *Session) Unlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.open && len(s.key) > 0
}

// Snapshot copies the key for a single encrypt. ok is false when locked.
func (s *Session) Snapshot() (key []byte, keyID string, kdf KDFParams, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.open || len(s.key) == 0 {
		return nil, "", KDFParams{}, false
	}
	return append([]byte(nil), s.key...), s.id, cloneKDF(s.kdf), true
}

func cloneKDF(p KDFParams) KDFParams {
	p.Salt = append([]byte(nil), p.Salt...)
	return p
}
