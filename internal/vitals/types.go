package vitals

import "time"

// API payload attached to GET /api/v1/overview.

type Snapshot struct {
	CollectedAt   time.Time      `json:"collected_at"`
	Stale         bool           `json:"stale"`
	Collecting    bool           `json:"collecting"`
	Scarce        string         `json:"scarce_resource,omitempty"` // memory | disk | cpu
	Machine       MachineVitals  `json:"machine"`
	Unaccounted   Unaccounted    `json:"unaccounted"`
	Rows          []RowVitals    `json:"rows"`
}

type MachineVitals struct {
	MemoryUsed      int64         `json:"memory_used_bytes"`
	MemoryTotal     int64         `json:"memory_total_bytes"`
	DiskUsed        int64         `json:"disk_used_bytes"`
	DiskTotal       int64         `json:"disk_total_bytes"`
	CPUPercent      float64       `json:"cpu_percent"`
	MemorySeries    []Point       `json:"memory_series"`
	DiskSeries      []Point       `json:"disk_series"`
	CPUSeries       []Point       `json:"cpu_series"`
	PressureMemory  bool          `json:"pressure_memory,omitempty"`
	PressureDisk    bool          `json:"pressure_disk,omitempty"`
	PressurePID     bool          `json:"pressure_pid,omitempty"`
}

type Unaccounted struct {
	MemoryBytes int64 `json:"memory_bytes"`
	DiskBytes   int64 `json:"disk_bytes"`
	CPUPercent  float64 `json:"cpu_percent"`
}

type RowVitals struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Kind         string `json:"kind"` // platform | instance
	InstanceType string `json:"type,omitempty"`
	Running      bool   `json:"running"`
	RankDriver   string `json:"rank_driver,omitempty"`
	Memory       Meter  `json:"memory"`
	CPU          Meter  `json:"cpu"`
	Disk         Meter  `json:"disk"`
	// Pod health from kubectl (latest sample).
	RunningSince          time.Time `json:"running_since,omitempty"`
	PodPhase              string    `json:"pod_phase,omitempty"`
	RestartCount          int       `json:"restart_count,omitempty"`
	LastTerminationReason string    `json:"last_termination_reason,omitempty"`
	PodReady              bool      `json:"pod_ready,omitempty"`
	WaitingReason         string    `json:"waiting_reason,omitempty"`
}

type Meter struct {
	UsedBytes    int64     `json:"used_bytes,omitempty"`
	TotalBytes   int64     `json:"total_bytes,omitempty"`
	Percent      float64   `json:"percent"`
	Known        bool      `json:"known"`
	ObservedAt   time.Time `json:"observed_at,omitempty"`
	NotRunning   bool      `json:"not_running,omitempty"`
	VolumeLabels []string  `json:"volume_labels,omitempty"`
	Series       []Point   `json:"series"`
}

type Point struct {
	At      time.Time `json:"at"`
	Percent float64   `json:"percent"`
}