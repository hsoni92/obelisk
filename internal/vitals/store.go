package vitals

import (
	"sync"
	"time"
)

const staleAfter = 3 * time.Minute

// Store holds vitals rings and the latest disk readings reused between ticks.
type Store struct {
	mu sync.RWMutex

	machine *Ring
	rows    map[string]*Ring

	// lastDisk per row key, reused when disk is not refreshed this tick.
	lastDisk map[string]diskHold

	collecting bool
	lastError  string

	diskQueue []diskJob
	diskIdx   int
}

type diskHold struct {
	Used     int64
	Total    int64
	Known    bool
	Observed time.Time
	Labels   []string
}

type diskJob struct {
	RowKey    string
	Release   string
	Mounts    []diskMount
}

type diskMount struct {
	Container string
	Path      string
	Label     string
}

func NewStore() *Store {
	return &Store{
		machine:  &Ring{},
		rows:     make(map[string]*Ring),
		lastDisk: make(map[string]diskHold),
	}
}

func (s *Store) SetCollecting(v bool) {
	s.mu.Lock()
	s.collecting = v
	s.mu.Unlock()
}

func (s *Store) ringFor(key string) *Ring {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rows[key]
	if !ok {
		r = &Ring{}
		s.rows[key] = r
	}
	return r
}

func (s *Store) AppendMachine(slot Slot) {
	s.mu.Lock()
	s.machine.Append(slot)
	s.mu.Unlock()
}

func (s *Store) AppendRow(key string, slot Slot) {
	s.ringFor(key).Append(slot)
}

func (s *Store) SetDiskHold(key string, h diskHold) {
	s.mu.Lock()
	s.lastDisk[key] = h
	s.mu.Unlock()
}

func (s *Store) DiskHold(key string) (diskHold, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.lastDisk[key]
	return h, ok
}

func (s *Store) SetDiskQueue(q []diskJob) {
	s.mu.Lock()
	s.diskQueue = q
	if s.diskIdx >= len(q) {
		s.diskIdx = 0
	}
	s.mu.Unlock()
}

func (s *Store) NextDiskJob() (diskJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.diskQueue) == 0 {
		return diskJob{}, false
	}
	j := s.diskQueue[s.diskIdx%len(s.diskQueue)]
	s.diskIdx++
	return j, true
}

func (s *Store) Snapshot(meta rowMeta) *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := &Snapshot{
		Collecting: s.collecting,
	}
	if latest, ok := s.machine.Latest(); ok {
		snap.CollectedAt = latest.At
		snap.Stale = time.Since(latest.At) > staleAfter
	}
	if snap.CollectedAt.IsZero() {
		snap.Collecting = true
	}

	snap.Machine = machineFromRing(s.machine)
	snap.Rows = buildRows(s.rows, meta, snap.Machine)
	snap.Scarce = scarceResource(snap.Machine)
	rankRows(snap.Rows, snap.Scarce)
	snap.Unaccounted = unaccounted(snap.Machine, snap.Rows)
	return snap
}

type rowMeta struct {
	keys []rowDef
}

type rowDef struct {
	Key       string
	Name      string
	Kind      string
	Type      string
	Running   bool
}

func machineFromRing(r *Ring) MachineVitals {
	ordered := r.Ordered()
	mv := MachineVitals{}
	if len(ordered) == 0 {
		return mv
	}
	last := ordered[len(ordered)-1]
	mv.MemoryUsed = last.MemoryUsed
	mv.MemoryTotal = last.MemoryTotal
	mv.DiskUsed = last.DiskUsed
	mv.DiskTotal = last.DiskTotal
	mv.CPUPercent = last.CPUSharePercent
	for _, sl := range ordered {
		mv.MemorySeries = append(mv.MemorySeries, pctPoint(sl.At, sl.MemoryUsed, sl.MemoryTotal, sl.MemoryKnown))
		mv.DiskSeries = append(mv.DiskSeries, pctPoint(sl.At, sl.DiskUsed, sl.DiskTotal, sl.DiskKnown))
		mv.CPUSeries = append(mv.CPUSeries, Point{At: sl.At, Percent: sl.CPUSharePercent})
	}
	return mv
}

func pctPoint(at time.Time, used, total int64, known bool) Point {
	if !known || total <= 0 {
		return Point{At: at, Percent: -1}
	}
	return Point{At: at, Percent: 100 * float64(used) / float64(total)}
}