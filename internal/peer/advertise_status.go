package peer

import (
	"sync"
	"time"
)

// AdvertiseStatus is the last outcome of a local→gateway catalog push (in-memory).
type AdvertiseStatus struct {
	OK            bool      `json:"ok"`
	LastOKAt      time.Time `json:"last_ok_at,omitempty"`
	LastAttemptAt time.Time `json:"last_attempt_at,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
}

var advertiseStatusMu sync.RWMutex
var lastAdvertiseStatus AdvertiseStatus

// RecordAdvertiseAttempt stores success or failure for the admin UI.
func RecordAdvertiseAttempt(err error) {
	advertiseStatusMu.Lock()
	defer advertiseStatusMu.Unlock()
	now := time.Now().UTC()
	lastAdvertiseStatus.LastAttemptAt = now
	if err == nil {
		lastAdvertiseStatus.OK = true
		lastAdvertiseStatus.LastOKAt = now
		lastAdvertiseStatus.LastError = ""
		return
	}
	lastAdvertiseStatus.OK = false
	lastAdvertiseStatus.LastError = err.Error()
}

// GetAdvertiseStatus returns a snapshot of the local advertiser loop.
func GetAdvertiseStatus() AdvertiseStatus {
	advertiseStatusMu.RLock()
	defer advertiseStatusMu.RUnlock()
	return lastAdvertiseStatus
}
