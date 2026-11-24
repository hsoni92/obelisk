package adminapi

import (
	"context"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const statesCacheTTL = 60 * time.Second

type statesCache struct {
	mu     sync.Mutex
	at     time.Time
	states map[string]string
}

func (s *Server) cachedServiceStates(ctx context.Context, spec obelisk.Spec, rec controller.Reconciler) map[string]string {
	s.statesCache.mu.Lock()
	defer s.statesCache.mu.Unlock()
	if s.statesCache.states != nil && time.Since(s.statesCache.at) < statesCacheTTL {
		return s.statesCache.states
	}
	states := rec.ObserveServiceStates(ctx, spec)
	s.statesCache.states = states
	s.statesCache.at = time.Now()
	return states
}
