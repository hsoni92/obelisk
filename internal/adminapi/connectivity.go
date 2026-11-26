package adminapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/history"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type connectivityPeer struct {
	ID                 string     `json:"id"`
	MachineID          string     `json:"machine_id,omitempty"`
	TailnetHost        string     `json:"tailnet_host,omitempty"`
	DuplicateMachineID bool       `json:"duplicate_machine_id"`
	LastSeen           *time.Time `json:"last_seen,omitempty"`
	HeartbeatFresh     bool       `json:"heartbeat_fresh"`
	ProbeKnown         bool       `json:"probe_known"`
	ProbeReachable     bool       `json:"probe_reachable"`
	ProbeCheckedAt     *time.Time `json:"probe_checked_at,omitempty"`
	ProbeError         string     `json:"probe_error,omitempty"`
	State              string     `json:"state"`
	StateSince         *time.Time `json:"state_since,omitempty"`
	// StateSinceCapped is true when the state started before the retained window,
	// so the displayed duration is a lower bound.
	StateSinceCapped bool     `json:"state_since_capped"`
	ObservedSeconds  int64    `json:"observed_seconds"`
	Strip            []string `json:"strip"`
}

type connectivityResponse struct {
	Role              string             `json:"role"`
	IsGateway         bool               `json:"is_gateway"`
	PeerViewEnabled   bool               `json:"peer_view_enabled"`
	WindowStart       time.Time          `json:"window_start"`
	WindowHours       int                `json:"window_hours"`
	BinMinutes        int                `json:"bin_minutes"`
	GatewayAliveAt    *time.Time         `json:"gateway_alive_at,omitempty"`
	MonitoringSince   *time.Time         `json:"monitoring_since,omitempty"`
	HeartbeatStaleSec int                `json:"heartbeat_stale_seconds"`
	ProbeIntervalSec  int                `json:"probe_interval_seconds"`
	Counts            map[string]int     `json:"counts"`
	Peers             []connectivityPeer `json:"peers"`
}

// handleConnectivity returns the gateway peer view: heartbeat and probe signals,
// the combined state, observed time in that state, and a 24-hour strip per peer.
// Local hosts get peer_view_enabled=false and an empty peer list.
func (s *Server) handleConnectivity(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	isGateway := obelisk.IsGateway(spec)
	resp := connectivityResponse{
		Role:              strings.TrimSpace(spec.Deployment.Role),
		IsGateway:         isGateway,
		PeerViewEnabled:   isGateway,
		WindowHours:       int(peer.StripWindow / time.Hour),
		BinMinutes:        int(peer.StripBin / time.Minute),
		HeartbeatStaleSec: int(peer.HeartbeatStaleAfter / time.Second),
		ProbeIntervalSec:  int(peer.ProbeInterval / time.Second),
		Counts:            map[string]int{},
		Peers:             []connectivityPeer{},
	}
	if !isGateway {
		writeJSON(w, resp)
		return
	}
	resp.WindowStart = time.Now().UTC().Add(-peer.StripWindow)

	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC()
	windowStart := now.Add(-peer.StripWindow)
	hist := s.overviewStore()
	events, err := hist.Since(time.Time{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastAlive := s.Connectivity.LastAlive()
	if !lastAlive.IsZero() {
		resp.GatewayAliveAt = timePtr(lastAlive)
	}
	coverage := peer.GatewayCoverage(events, windowStart, now, lastAlive)
	if len(coverage) > 0 {
		resp.MonitoringSince = timePtr(coverage[0].From)
	}

	records := reg.ListPeers()
	machineCount := map[string]int{}
	for _, p := range records {
		if id := strings.TrimSpace(p.MachineID); id != "" {
			machineCount[id]++
		}
	}
	for _, p := range records {
		probe := s.Connectivity.ProbeStatus(p.ID)
		reachable := true
		if probe.Known {
			reachable = probe.Reachable
		}
		state := peer.CombinedState(p.LastSeen, now, reachable)
		since, capped := stateSince(events, p.ID, state, coverage, windowStart, now)
		cp := connectivityPeer{
			ID:                 p.ID,
			MachineID:          strings.TrimSpace(p.MachineID),
			TailnetHost:        strings.TrimSpace(p.TailnetHost),
			DuplicateMachineID: strings.TrimSpace(p.MachineID) != "" && machineCount[strings.TrimSpace(p.MachineID)] > 1,
			HeartbeatFresh:     p.Reachable,
			ProbeKnown:         probe.Known,
			ProbeReachable:     reachable,
			ProbeError:         probe.LastError,
			State:              state,
			StateSince:         timePtr(since),
			StateSinceCapped:   capped,
			ObservedSeconds:    int64(peer.ObservedWithin(coverage, since, now) / time.Second),
		}
		if !p.LastSeen.IsZero() {
			cp.LastSeen = timePtr(p.LastSeen)
		}
		if probe.Known {
			cp.ProbeCheckedAt = timePtr(probe.CheckedAt)
		}
		segs := peer.PeerSegments(events, peer.PeerHistoryKey(p.ID), windowStart, now, state)
		cp.Strip = peer.StripBins(segs, coverage, windowStart, peer.StripBinsNum, peer.StripBin)
		resp.Counts[state]++
		resp.Peers = append(resp.Peers, cp)
	}
	writeJSON(w, resp)
}

// stateSince returns when the peer's current state began, and whether that
// start is older than the retained window. The latest transition event for the
// peer gives the start. Without one, observation began at the coverage start.
func stateSince(events []history.Event, peerID, state string, coverage []peer.Span, windowStart, now time.Time) (time.Time, bool) {
	key := peer.PeerHistoryKey(peerID)
	var latest *history.Event
	for i := range events {
		e := &events[i]
		if e.Kind != peer.KindPeer || e.Key != key {
			continue
		}
		if latest == nil || e.At.After(latest.At) {
			latest = e
		}
	}
	if latest != nil && latest.State == state {
		return latest.At, false
	}
	if len(coverage) > 0 {
		from := coverage[0].From
		return from, from.Equal(windowStart)
	}
	return now, false
}

func timePtr(t time.Time) *time.Time {
	return &t
}
