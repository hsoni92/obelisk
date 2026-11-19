package controller

import (
	"sync"
	"time"
)

// Reason codes surfaced in overview/API when an instance is not running cleanly.
const (
	ReasonReleaseMissing    = "release_missing"
	ReasonHelmPendingInstall = "helm_pending_install"
	ReasonHelmPendingUpgrade = "helm_pending_upgrade"
	ReasonHelmPendingRollback = "helm_pending_rollback"
	ReasonHelmFailed        = "helm_failed"
	ReasonPodCrashLoop      = "pod_crashloop"
	ReasonPodImagePull      = "pod_image_pull"
	ReasonPodPendingPVC     = "pod_pending_pvc"
	ReasonPodNotReady       = "pod_not_ready"
	ReasonDBInitFailed      = "db_init_failed"
	ReasonChartInvalid      = "chart_invalid"
	ReasonApplyTimeout      = "apply_timeout"
	ReasonBackoffWait       = "backoff_wait"
	ReasonStoppingDrain     = "stopping_drain"
	ReasonInitialInstall    = "initial_install_pending"
)

type InstanceStatus struct {
	State       string
	ReasonCode  string
	Detail      string
	DesiredHash uint64
	Attempts    int
	LastAttempt time.Time
	LastSuccess time.Time
	NextRetry   time.Time
	PendingSince time.Time
}

type statusStore struct {
	mu   sync.RWMutex
	byKey map[string]*InstanceStatus
}

func newStatusStore() *statusStore {
	return &statusStore{byKey: map[string]*InstanceStatus{}}
}

func (s *statusStore) get(key string) *InstanceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.byKey[key]
	if st == nil {
		return nil
	}
	cp := *st
	return &cp
}

func (s *statusStore) set(key string, st InstanceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[key] = &st
}

func (s *statusStore) states(keys []string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		if st, ok := s.byKey[k]; ok && st.State != "" {
			out[k] = st.State
		}
	}
	return out
}