package adminapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/history"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// OverviewComponent is one platform or data-service row.
type OverviewComponent struct {
	Key        string `json:"key"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	Desired    bool   `json:"desired"`
	State      string `json:"state"`
	HelmStatus string `json:"helm_status,omitempty"`
	Revision   int    `json:"revision,omitempty"`
	Reason     string `json:"reason,omitempty"`
	PodsReady  int    `json:"pods_ready,omitempty"`
	PodsTotal  int    `json:"pods_total,omitempty"`
	Release    string `json:"release"`
}

type overviewSummary struct {
	Running  int `json:"running"`
	Stopped  int `json:"stopped"`
	Pending  int `json:"pending"`
	Degraded int `json:"degraded"`
	Other    int `json:"other"`
}

type overviewResponse struct {
	GeneratedAt   time.Time           `json:"generated_at"`
	ReconcileMode string              `json:"reconcileMode"`
	Profile       string              `json:"profile"`
	Tailscale     bool                `json:"tailscale"`
	MachineID     string              `json:"machineId"`
	BackupReady   bool                `json:"backup_ready"`
	Components    []OverviewComponent `json:"components"`
	Summary       overviewSummary     `json:"summary"`
	History       []history.Event     `json:"history"`
	HistoryHours  float64             `json:"history_hours"`
}

func (s *Server) overviewStore() *history.Store {
	return history.NewStore(config.HistoryPath(config.EnsureReadablePath(s.ConfigPath)))
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hours := 6.0
	if q := r.URL.Query().Get("hours"); q != "" {
		if v, err := strconv.ParseFloat(q, 64); err == nil && v > 0 && v <= 24 {
			hours = v
		}
	}

	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	states := rec.ObserveServiceStates(r.Context(), spec)
	simulate := rec.SimulateMode()

	components := []OverviewComponent{}
	sum := overviewSummary{}

	// Platform release
	platform := s.observeRelease(r, rec, OverviewComponent{
		Key:     "platform/obelisk-platform",
		Kind:    "platform",
		Name:    "Platform",
		Desired: true,
		State:   "running",
		Release: controller.PlatformRelease,
	}, simulate)
	components = append(components, platform)
	bumpSummary(&sum, platform.State)

	for _, inst := range spec.Instances {
		st := states[inst.StateKey()]
		if st == "" {
			st = "loading"
		}
		comp := OverviewComponent{
			Key:        inst.StateKey(),
			Kind:       "instance",
			Name:       instanceDisplayName(inst),
			Type:       inst.Type,
			InstanceID: inst.ID,
			Desired:    inst.Enabled,
			State:      st,
			Release:    inst.ReleaseName(),
		}
		if inst.Enabled {
			comp = s.observeRelease(r, rec, comp, simulate)
		} else {
			comp.State = "stopped"
			comp.Reason = "disabled in desired state"
		}
		components = append(components, comp)
		bumpSummary(&sum, comp.State)
	}

	store := s.overviewStore()
	for _, c := range components {
		detail := c.Reason
		if c.PodsTotal > 0 {
			detail = detail + " pods " + strconv.Itoa(c.PodsReady) + "/" + strconv.Itoa(c.PodsTotal)
		}
		_ = store.Record(history.Event{
			Key:        c.Key,
			Kind:       c.Kind,
			State:      c.State,
			Detail:     detail,
			HelmStatus: c.HelmStatus,
		})
	}

	since := time.Now().UTC().Add(-time.Duration(hours * float64(time.Hour)))
	events, _ := store.Since(since)

	bt := s.buildBackupTarget(cfg, spec)
	resp := overviewResponse{
		GeneratedAt:   time.Now().UTC(),
		ReconcileMode: modeLabel(simulate),
		Profile:       spec.Deployment.Profile,
		Tailscale:     spec.Deployment.Tailscale.Enabled,
		MachineID:     spec.BackupBucketName(),
		BackupReady:   bt.BackupReady,
		Components:    components,
		Summary:       sum,
		History:       events,
		HistoryHours:  hours,
	}
	writeJSON(w, resp)
}

func modeLabel(simulate bool) string {
	if simulate {
		return "simulate"
	}
	return "cluster"
}

func (s *Server) observeRelease(r *http.Request, rec controller.Reconciler, comp OverviewComponent, simulate bool) OverviewComponent {
	if simulate {
		if comp.Desired {
			comp.State = "running"
		}
		comp.HelmStatus = "deployed"
		comp.Revision = 1
		comp.PodsReady = 1
		comp.PodsTotal = 1
		return comp
	}
	if !comp.Desired {
		comp.State = "stopped"
		return comp
	}
	helm := rec.HelmReleaseDetail(r.Context(), comp.Release)
	if helm.Err != nil {
		comp.State = "degraded"
		comp.Reason = helm.Err.Error()
		return comp
	}
	comp.HelmStatus = helm.Status
	comp.Revision = helm.Revision
	if helm.Description != "" {
		comp.Reason = helm.Description
	}
	switch helm.Status {
	case "deployed":
		// keep ObserveServiceStates mapping
	case "failed", "uninstalling":
		comp.State = "degraded"
		if comp.Reason == "" {
			comp.Reason = "helm status: " + helm.Status
		}
	case "pending-install", "pending-upgrade", "pending-rollback":
		comp.State = "pending"
	default:
		if comp.State == "" {
			comp.State = helm.Status
		}
	}
	pods := rec.PodsForRelease(r.Context(), comp.Release)
	if pods.Err == nil {
		comp.PodsReady = pods.Ready
		comp.PodsTotal = pods.Total
		if comp.Desired && comp.State == "running" && pods.Total > 0 && pods.Ready < pods.Total {
			comp.State = "degraded"
			if comp.Reason == "" {
				comp.Reason = "pods not ready"
			}
		}
	}
	return comp
}

func bumpSummary(s *overviewSummary, state string) {
	switch state {
	case "running":
		s.Running++
	case "stopped":
		s.Stopped++
	case "pending":
		s.Pending++
	case "degraded":
		s.Degraded++
	default:
		s.Other++
	}
}

func instanceDisplayName(inst obelisk.ServiceInstance) string {
	if inst.DisplayName != "" {
		return inst.DisplayName
	}
	return inst.ID
}