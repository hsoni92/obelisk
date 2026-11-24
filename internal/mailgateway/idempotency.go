package mailgateway

import (
	"sync"
	"time"
)

type idemEntry struct {
	accountID string
	from      string
	expires   time.Time
}

type idempotency struct {
	mu      sync.Mutex
	entries map[string]idemEntry
}

func newIdempotency() *idempotency {
	return &idempotency{entries: map[string]idemEntry{}}
}

func (i *idempotency) get(key string) (idemEntry, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.pruneLocked()
	e, ok := i.entries[key]
	return e, ok
}

func (i *idempotency) put(key, accountID, from string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.entries[key] = idemEntry{
		accountID: accountID,
		from:      from,
		expires:   time.Now().Add(24 * time.Hour),
	}
}

func (i *idempotency) pruneLocked() {
	now := time.Now()
	for k, v := range i.entries {
		if now.After(v.expires) {
			delete(i.entries, k)
		}
	}
}
