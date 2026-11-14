package openroutergateway

import "sync"

type upstreamAccount struct {
	ID  string
	Key string
}

// Selector round-robins across enabled accounts with upstream keys.
type Selector struct {
	mu       sync.Mutex
	next     int
	accounts []upstreamAccount
}

func NewSelector(accounts []upstreamAccount) *Selector {
	cp := make([]upstreamAccount, len(accounts))
	copy(cp, accounts)
	return &Selector{accounts: cp}
}

func (s *Selector) Len() int {
	return len(s.accounts)
}

func (s *Selector) Snapshot() []upstreamAccount {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]upstreamAccount, len(s.accounts))
	copy(out, s.accounts)
	return out
}

func (s *Selector) Next() (upstreamAccount, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.accounts) == 0 {
		return upstreamAccount{}, false
	}
	acc := s.accounts[s.next%len(s.accounts)]
	s.next++
	return acc, true
}