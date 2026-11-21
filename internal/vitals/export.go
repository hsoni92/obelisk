package vitals

import (
	"math"
	"sort"
	"time"
)

// validRunningSince rejects unset/zero pod start times that JSON encodes as year 0001.
func validRunningSince(t time.Time) (time.Time, bool) {
	if t.IsZero() || t.Year() < 2000 {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func scarceResource(m MachineVitals) string {
	mem := sharePct(m.MemoryUsed, m.MemoryTotal)
	disk := sharePct(m.DiskUsed, m.DiskTotal)
	cpu := m.CPUPercent
	if m.MemoryTotal <= 0 && m.DiskTotal <= 0 && cpu <= 0 {
		return ""
	}
	if mem >= disk && mem >= cpu {
		return "memory"
	}
	if disk >= mem && disk >= cpu {
		return "disk"
	}
	return "cpu"
}

func sharePct(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return 100 * float64(used) / float64(total)
}

func buildRows(rings map[string]*Ring, meta rowMeta, machine MachineVitals) []RowVitals {
	out := make([]RowVitals, 0, len(meta.keys))
	for _, def := range meta.keys {
		r := rings[def.Key]
		row := RowVitals{
			Key:          def.Key,
			Name:         def.Name,
			Kind:         def.Kind,
			InstanceType: def.Type,
			Running:      def.Running,
		}
		if r == nil || r.Count() == 0 {
			row.Memory = Meter{Known: false, Series: nil}
			row.CPU = Meter{Known: false}
			row.Disk = Meter{Known: false}
			out = append(out, row)
			continue
		}
		latest, hasLatest := r.Latest()
		ordered := r.Ordered()
		row.Memory = meterFromRing(ordered, machine.MemoryTotal, def.Running, "memory")
		row.CPU = cpuMeterFromRing(ordered, def.Running)
		row.Disk = meterFromRing(ordered, machine.DiskTotal, def.Running, "disk")
		if hasLatest {
			if t, ok := validRunningSince(latest.RunningSince); ok {
				row.RunningSince = t
			}
			row.PodPhase = latest.PodPhase
			row.RestartCount = latest.RestartCount
			row.LastTerminationReason = latest.LastTerminationReason
			row.PodReady = latest.PodReady
			row.WaitingReason = latest.WaitingReason
		}
		if !def.Running {
			row.Memory.NotRunning = true
			row.CPU.NotRunning = true
			row.Memory.Percent = 0
			row.CPU.Percent = 0
			row.Memory.Known = true
			row.CPU.Known = true
		}
		out = append(out, row)
	}
	return out
}

func meterFromRing(ordered []Slot, machineTotal int64, running bool, which string) Meter {
	m := Meter{Series: make([]Point, 0, len(ordered))}
	if len(ordered) == 0 {
		return m
	}
	last := ordered[len(ordered)-1]
	var used int64
	var known bool
	var obs time.Time
	for _, sl := range ordered {
		var u int64
		var k bool
		switch which {
		case "memory":
			u, k = sl.MemoryUsed, sl.MemoryKnown
		case "disk":
			u, k = sl.DiskUsed, sl.DiskKnown
			if !sl.DiskObserved.IsZero() {
				obs = sl.DiskObserved
			}
		}
		m.Series = append(m.Series, pctPoint(sl.At, u, machineTotal, k && machineTotal > 0))
	}
	switch which {
	case "memory":
		used, known = last.MemoryUsed, last.MemoryKnown
	case "disk":
		used, known = last.DiskUsed, last.DiskKnown
		if !last.DiskObserved.IsZero() {
			obs = last.DiskObserved
		}
	}
	if machineTotal > 0 && known {
		m.UsedBytes = used
		m.TotalBytes = machineTotal
		m.Percent = sharePct(used, machineTotal)
		m.Known = true
		m.ObservedAt = last.At
		if which == "disk" && !obs.IsZero() {
			m.ObservedAt = obs
		}
	} else if !running && which == "disk" && known {
		m.UsedBytes = used
		m.TotalBytes = machineTotal
		m.Percent = sharePct(used, machineTotal)
		m.Known = true
		m.ObservedAt = obs
	}
	return m
}

func cpuMeterFromRing(ordered []Slot, running bool) Meter {
	m := Meter{Series: make([]Point, 0, len(ordered))}
	if len(ordered) == 0 {
		return m
	}
	last := ordered[len(ordered)-1]
	for _, sl := range ordered {
		if sl.CPUKnown {
			m.Series = append(m.Series, Point{At: sl.At, Percent: sl.CPUSharePercent})
		} else {
			m.Series = append(m.Series, Point{At: sl.At, Percent: -1})
		}
	}
	if last.CPUKnown {
		m.Percent = last.CPUSharePercent
		m.Known = true
		m.ObservedAt = last.At
	}
	_ = running
	return m
}

func rankRows(rows []RowVitals, scarce string) {
	type scored struct {
		row   RowVitals
		score float64
		unk   bool
	}
	scoredRows := make([]scored, 0, len(rows))
	for _, row := range rows {
		sc, unk := rowScore(row, scarce)
		scoredRows = append(scoredRows, scored{row: row, score: sc, unk: unk})
	}
	sort.SliceStable(scoredRows, func(i, j int) bool {
		if scoredRows[i].unk != scoredRows[j].unk {
			return scoredRows[i].unk
		}
		if math.Abs(scoredRows[i].score-scoredRows[j].score) > 0.001 {
			return scoredRows[i].score > scoredRows[j].score
		}
		return scoredRows[i].row.Key < scoredRows[j].row.Key
	})
	for i := range scoredRows {
		r := scoredRows[i].row
		r.RankDriver = scarce
		scoredRows[i].row = r
	}
	for i, sc := range scoredRows {
		rows[i] = sc.row
	}
}

func rowScore(row RowVitals, scarce string) (float64, bool) {
	switch scarce {
	case "memory":
		if !row.Running {
			return 0, false
		}
		if !row.Memory.Known {
			return 0, true
		}
		return row.Memory.Percent, false
	case "disk":
		if !row.Disk.Known {
			return 0, true
		}
		return row.Disk.Percent, false
	default:
		if !row.Running {
			return 0, false
		}
		if !row.CPU.Known {
			return 0, true
		}
		return row.CPU.Percent, false
	}
}

func unaccounted(m MachineVitals, rows []RowVitals) Unaccounted {
	var mem, disk int64
	var cpu float64
	for _, r := range rows {
		if r.Memory.Known && r.Running {
			mem += r.Memory.UsedBytes
		}
		if r.Disk.Known {
			disk += r.Disk.UsedBytes
		}
		if r.CPU.Known && r.Running {
			cpu += r.CPU.Percent
		}
	}
	u := Unaccounted{}
	if m.MemoryTotal > 0 && m.MemoryUsed > mem {
		u.MemoryBytes = m.MemoryUsed - mem
	}
	if m.DiskTotal > 0 && m.DiskUsed > disk {
		u.DiskBytes = m.DiskUsed - disk
	}
	if m.CPUPercent > cpu {
		u.CPUPercent = m.CPUPercent - cpu
	}
	return u
}