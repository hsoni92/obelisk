package controller

import (
	"sync"
	"time"
)

const orphanGraceScans = 3

var (
	deleteIntentMu sync.Mutex
	deleteIntent   = map[string]time.Time{}
	orphanScansMu  sync.Mutex
	orphanScans    = map[string]int{}
)

// MarkReleaseDeleteIntent records that a release was explicitly removed from spec and may be pruned.
func MarkReleaseDeleteIntent(release string) {
	deleteIntentMu.Lock()
	deleteIntent[release] = time.Now()
	deleteIntentMu.Unlock()
}

func consumeDeleteIntent(release string) bool {
	deleteIntentMu.Lock()
	defer deleteIntentMu.Unlock()
	if _, ok := deleteIntent[release]; ok {
		delete(deleteIntent, release)
		return true
	}
	return false
}

func orphanGraceReached(release string) bool {
	orphanScansMu.Lock()
	defer orphanScansMu.Unlock()
	orphanScans[release]++
	return orphanScans[release] >= orphanGraceScans
}

func resetOrphanGrace(release string) {
	orphanScansMu.Lock()
	delete(orphanScans, release)
	orphanScansMu.Unlock()
}

func clearOrphanTracking(removed []string) {
	orphanScansMu.Lock()
	defer orphanScansMu.Unlock()
	for _, rel := range removed {
		delete(orphanScans, rel)
	}
}