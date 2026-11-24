package diagnostics

import "time"

// InstanceRef selects one enabled instance row.
type InstanceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// RunResponse is the outcome of POST /api/v1/diagnostics/run.
type RunResponse struct {
	StartedAt string           `json:"started_at"`
	Results   []InstanceResult `json:"results"`
}

// InstanceResult is a single end-to-end probe using published connection URLs.
type InstanceResult struct {
	Type          string  `json:"type"`
	ID            string  `json:"id"`
	DisplayName   string  `json:"display_name,omitempty"`
	ClusterState  string  `json:"cluster_state"`
	OK            bool    `json:"ok"`
	Skipped       bool    `json:"skipped,omitempty"`
	SkippedReason string  `json:"skipped_reason,omitempty"`
	DurationMS    int64   `json:"duration_ms"`
	Summary       string  `json:"summary"`
	PublicHost    string  `json:"public_host,omitempty"`
	Checks        []Check `json:"checks,omitempty"`
}

// Check is one step inside an instance probe.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
