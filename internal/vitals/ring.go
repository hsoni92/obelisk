package vitals

import (
	"sync"
	"time"
)

const RingSize = 30

// Slot is one minute sample for a workload or the machine.
type Slot struct {
	At time.Time `json:"at"`

	MemoryUsed  int64 `json:"memory_used_bytes"`
	MemoryTotal int64 `json:"memory_total_bytes"`
	MemoryKnown bool  `json:"memory_known"`

	CPUSharePercent float64 `json:"cpu_share_percent"`
	CPUKnown        bool    `json:"cpu_known"`

	DiskUsed     int64     `json:"disk_used_bytes"`
	DiskTotal    int64     `json:"disk_total_bytes"`
	DiskKnown    bool      `json:"disk_known"`
	DiskObserved time.Time `json:"disk_observed_at"`

	RestartCount          int       `json:"restart_count,omitempty"`
	LastTerminationReason string    `json:"last_termination_reason,omitempty"`
	PodPhase              string    `json:"pod_phase,omitempty"`
	RunningSince          time.Time `json:"running_since,omitempty"`
	PodReady              bool      `json:"pod_ready,omitempty"`
	WaitingReason         string    `json:"waiting_reason,omitempty"`
}

// Ring is a fixed-size circular buffer of slots.
type Ring struct {
	mu    sync.RWMutex
	slots [RingSize]Slot
	n     int
	head  int
}

func (r *Ring) Append(s Slot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.slots[r.head] = s
	r.head = (r.head + 1) % RingSize
	if r.n < RingSize {
		r.n++
	}
}

func (r *Ring) Latest() (Slot, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.n == 0 {
		return Slot{}, false
	}
	idx := (r.head - 1 + RingSize) % RingSize
	return r.slots[idx], true
}

func (r *Ring) Ordered() []Slot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.n == 0 {
		return nil
	}
	out := make([]Slot, 0, r.n)
	start := (r.head - r.n + RingSize) % RingSize
	for i := 0; i < r.n; i++ {
		out = append(out, r.slots[(start+i)%RingSize])
	}
	return out
}

func (r *Ring) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.n
}
