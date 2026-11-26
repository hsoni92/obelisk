package peer

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/history"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const (
	// MonitorTick is how often the gateway evaluates heartbeats and writes its liveness marker.
	MonitorTick = 30 * time.Second
	// ProbeInterval is the time between reachability probes for one peer.
	ProbeInterval = 60 * time.Second
	// ProbeTimeout matches the nginx connect timeout used for federated routes.
	ProbeTimeout = 5 * time.Second
	// ProbeConcurrency caps simultaneous probes so many peers do not open many connections at once.
	ProbeConcurrency = 4
	// ProbeJitter spreads probes for different peers so they do not align.
	ProbeJitter = 10 * time.Second
	// ProbeFailuresToDegrade is the consecutive failures needed before a peer is marked degraded.
	ProbeFailuresToDegrade = 2
	// ProbeSuccessesToRecover is the consecutive successes needed before a degraded peer recovers.
	ProbeSuccessesToRecover = 2
	// GatewayGapAfter is how stale the liveness marker may be before a gap is recorded.
	GatewayGapAfter = 2 * time.Minute
	// GatewayCheckpointEvery records an "up" event this often so a window never loses its start.
	GatewayCheckpointEvery = time.Hour
	// MaxTrackedPeers bounds monitoring work on one gateway.
	MaxTrackedPeers = 50

	gatewayMarkerFile = "gateway.alive"
)

// ProbeFunc checks that host:port accepts a TCP connection within ctx's deadline.
type ProbeFunc func(ctx context.Context, host string, port int) error

// TCPProbe is the production probe: a bare TCP connect, no data sent.
// It proves the path and port are open, not that the service behind it is healthy.
func TCPProbe(ctx context.Context, host string, port int) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	return conn.Close()
}

type probeState struct {
	reachable  bool
	failStreak int
	okStreak   int
	inFlight   bool
	nextProbe  time.Time
	checkedAt  time.Time
	lastErr    string
}

// ProbeStatus is the reachability signal for one peer, as last measured.
type ProbeStatus struct {
	Known     bool
	Reachable bool
	CheckedAt time.Time
	LastError string
}

// Monitor runs on a gateway. It derives each peer's state from its heartbeat and
// a TCP reachability probe, writes state transitions to the history store, and
// keeps the gateway liveness marker that separates "peer down" from "gateway down".
type Monitor struct {
	configPath string
	hist       *history.Store
	markerPath string
	probe      ProbeFunc
	now        func() time.Time
	sem        chan struct{}

	mu          sync.Mutex
	probes      map[string]*probeState
	states      map[string]string
	lastAlive   time.Time
	aliveLoaded bool
	lastCheck   time.Time
}

// NewMonitor builds a monitor for the config at configPath. Call Run to start it.
func NewMonitor(configPath string, hist *history.Store) *Monitor {
	cfg := config.EnsureReadablePath(configPath)
	return &Monitor{
		configPath: cfg,
		hist:       hist,
		markerPath: filepath.Join(filepath.Dir(config.HistoryPath(cfg)), gatewayMarkerFile),
		probe:      TCPProbe,
		now:        func() time.Time { return time.Now().UTC() },
		sem:        make(chan struct{}, ProbeConcurrency),
		probes:     map[string]*probeState{},
		states:     map[string]string{},
	}
}

// StartMonitor runs a monitor in the background until ctx is done.
func StartMonitor(ctx context.Context, configPath string, hist *history.Store) *Monitor {
	m := NewMonitor(configPath, hist)
	go m.Run(ctx)
	return m
}

// Run ticks until ctx is done.
func (m *Monitor) Run(ctx context.Context) {
	t := time.NewTicker(MonitorTick)
	defer t.Stop()
	m.Tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.Tick(ctx)
		}
	}
}

// Tick runs one evaluation. It does nothing unless this host is a gateway,
// because the role can change at runtime.
func (m *Monitor) Tick(ctx context.Context) {
	now := m.now()
	spec, err := config.Load(m.configPath)
	if err != nil || !obelisk.IsGateway(spec) {
		return
	}
	m.recordLiveness(now)

	reg, err := LoadRegistry(m.configPath)
	if err != nil {
		// Keep the last evaluated states. Do not treat a read failure as peers going down.
		slog.Warn("connectivity: peer registry unavailable", "err", err)
		return
	}
	m.evaluate(ctx, now, reg.ListPeers())
}

// recordLiveness writes the marker and records gap and checkpoint events.
func (m *Monitor) recordLiveness(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.aliveLoaded {
		m.lastAlive = readMarker(m.markerPath)
		m.aliveLoaded = true
	}
	prev := m.lastAlive
	switch {
	case prev.IsZero():
		m.append(history.Event{At: now, Kind: KindGateway, Key: KindGateway, State: GatewayUp})
		m.lastCheck = now
	case now.Sub(prev) > GatewayGapAfter:
		m.append(history.Event{At: prev, Until: now, Kind: KindGateway, Key: KindGateway, State: GatewayGap})
		m.append(history.Event{At: now, Kind: KindGateway, Key: KindGateway, State: GatewayUp})
		m.lastCheck = now
	case now.Sub(m.lastCheck) >= GatewayCheckpointEvery:
		m.append(history.Event{At: now, Kind: KindGateway, Key: KindGateway, State: GatewayUp})
		m.lastCheck = now
	}
	m.lastAlive = now
	if err := writeMarker(m.markerPath, now); err != nil {
		slog.Warn("connectivity: write gateway marker", "err", err)
	}
}

func (m *Monitor) append(e history.Event) {
	if m.hist == nil {
		return
	}
	if err := m.hist.Append(e); err != nil {
		slog.Warn("connectivity: write gateway event", "state", e.State, "err", err)
	}
}

// evaluate updates probe scheduling, combined states, and transition events.
func (m *Monitor) evaluate(ctx context.Context, now time.Time, peers []PeerRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()

	seen := map[string]bool{}
	for _, p := range peers {
		if len(seen) >= MaxTrackedPeers {
			break
		}
		id := p.ID
		seen[id] = true
		ps := m.probes[id]
		if ps == nil {
			ps = &probeState{reachable: true, nextProbe: now}
			m.probes[id] = ps
		}
		fresh := heartbeatFresh(p.LastSeen, now, HeartbeatStaleAfter)
		if fresh && !ps.inFlight && !now.Before(ps.nextProbe) {
			ps.inFlight = true
			ps.nextProbe = now.Add(ProbeInterval + time.Duration(rand.Int64N(int64(ProbeJitter))))
			go m.runProbe(ctx, id, probeHost(p), probePort(p))
		}
		state := CombinedState(p.LastSeen, now, ps.reachable)
		if state != m.states[id] {
			m.states[id] = state
			detail := ""
			if state == StateDegraded {
				detail = ps.lastErr
			}
			if m.hist != nil {
				if err := m.hist.Record(history.Event{
					At: now, Key: PeerHistoryKey(id), Kind: KindPeer, State: state, Detail: detail,
				}); err != nil {
					slog.Warn("connectivity: record peer state", "peer", id, "err", err)
				}
			}
		}
	}
	for id := range m.probes {
		if !seen[id] {
			delete(m.probes, id)
			delete(m.states, id)
		}
	}
}

func probeHost(p PeerRecord) string {
	return strings.TrimSpace(p.TailnetHost)
}

func probePort(p PeerRecord) int {
	if p.HTTPBackhaulPort > 0 {
		return p.HTTPBackhaulPort
	}
	return obelisk.AdminHTTPPort()
}

// runProbe runs one probe and applies hysteresis. Transitions are written on the next tick.
func (m *Monitor) runProbe(ctx context.Context, id, host string, port int) {
	select {
	case m.sem <- struct{}{}:
	case <-ctx.Done():
		m.finishProbe(id, nil, true)
		return
	}
	defer func() { <-m.sem }()

	var err error
	if host == "" {
		err = fmt.Errorf("peer has no tailnet host")
	} else {
		pctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
		err = m.probe(pctx, host, port)
		cancel()
	}
	m.finishProbe(id, err, false)
}

func (m *Monitor) finishProbe(id string, err error, cancelled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.probes[id]
	if ps == nil {
		return
	}
	ps.inFlight = false
	if cancelled {
		return
	}
	now := m.now()
	ps.checkedAt = now
	if err != nil {
		ps.okStreak = 0
		ps.failStreak++
		ps.lastErr = err.Error()
		if ps.failStreak >= ProbeFailuresToDegrade {
			ps.reachable = false
		}
		return
	}
	ps.failStreak = 0
	ps.lastErr = ""
	ps.okStreak++
	if ps.okStreak >= ProbeSuccessesToRecover {
		ps.reachable = true
	}
}

// ProbeStatus returns the latest probe result for a peer.
func (m *Monitor) ProbeStatus(peerID string) ProbeStatus {
	if m == nil {
		return ProbeStatus{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.probes[peerID]
	if ps == nil || ps.checkedAt.IsZero() {
		return ProbeStatus{}
	}
	return ProbeStatus{Known: true, Reachable: ps.reachable, CheckedAt: ps.checkedAt, LastError: ps.lastErr}
}

// Reachable reports the hysteresis-filtered probe result, true when not yet measured.
func (m *Monitor) Reachable(peerID string) bool {
	if m == nil {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if ps := m.probes[peerID]; ps != nil {
		return ps.reachable
	}
	return true
}

// LastAlive returns the most recent gateway liveness time, or zero if unknown.
func (m *Monitor) LastAlive() time.Time {
	if m == nil {
		return time.Time{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastAlive
}

func readMarker(path string) time.Time {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}
	}
	return t
}

// writeMarker replaces the liveness marker atomically.
func writeMarker(path string, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), gatewayMarkerFile+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.WriteString(now.UTC().Format(time.RFC3339Nano) + "\n"); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
