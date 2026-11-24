package mailgateway

import "sync"

// Selector round-robins across enabled mail accounts.
type Selector struct {
	mu       sync.Mutex
	next     int
	accounts []mailAccount
}

func NewSelector(accounts []mailAccount) *Selector {
	cp := make([]mailAccount, len(accounts))
	copy(cp, accounts)
	return &Selector{accounts: cp}
}

func (s *Selector) Len() int {
	return len(s.accounts)
}

func (s *Selector) OrderedFrom(start int) []mailAccount {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.accounts) == 0 {
		return nil
	}
	start = start % len(s.accounts)
	out := make([]mailAccount, len(s.accounts))
	copy(out, s.accounts[start:])
	copy(out[len(s.accounts)-start:], s.accounts[:start])
	return out
}

func (s *Selector) Next() (mailAccount, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.accounts) == 0 {
		return mailAccount{}, false
	}
	acc := s.accounts[s.next%len(s.accounts)]
	s.next++
	return acc, true
}

func (s *Selector) ByID(id string) (mailAccount, bool) {
	for _, a := range s.accounts {
		if a.ID == id {
			return a, true
		}
	}
	return mailAccount{}, false
}

// RotateList returns accounts in round-robin order and advances the cursor.
func (s *Selector) RotateList() []mailAccount {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.accounts) == 0 {
		return nil
	}
	start := s.next % len(s.accounts)
	s.next++
	out := make([]mailAccount, len(s.accounts))
	copy(out, s.accounts[start:])
	copy(out[len(s.accounts)-start:], s.accounts[:start])
	return out
}
