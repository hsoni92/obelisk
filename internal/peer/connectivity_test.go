package peer

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/history"
)

var t0 = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

func TestCombinedState(t *testing.T) {
	now := t0
	cases := []struct {
		name      string
		lastSeen  time.Time
		reachable bool
		want      string
	}{
		{"never advertised", time.Time{}, true, StateNever},
		{"stale heartbeat", now.Add(-3 * time.Minute), true, StateOffline},
		{"fresh but probe failing", now.Add(-10 * time.Second), false, StateDegraded},
		{"fresh and reachable", now.Add(-10 * time.Second), true, StateConnected},
	}
	for _, c := range cases {
		if got := CombinedState(c.lastSeen, now, c.reachable); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestGatewayCoverageGapIsNotObserved(t *testing.T) {
	now := t0.Add(3 * time.Hour)
	events := []history.Event{
		{Kind: KindGateway, State: GatewayUp, At: t0},
		{Kind: KindGateway, State: GatewayGap, At: t0.Add(1 * time.Hour), Until: t0.Add(2 * time.Hour)},
		{Kind: KindGateway, State: GatewayUp, At: t0.Add(2 * time.Hour)},
	}
	lastAlive := now
	cov := GatewayCoverage(events, t0, now, lastAlive)
	if got := ObservedWithin(cov, t0, now); got != 2*time.Hour {
		t.Fatalf("observed: got %v want 2h", got)
	}
	if anyOverlap(cov, t0.Add(90*time.Minute), t0.Add(105*time.Minute)) {
		t.Fatal("gap window must not be covered")
	}
}

func TestGatewayCoverageAssumesRunningAfterPrune(t *testing.T) {
	// The "up" event fell out of retention; the marker shows the gateway is alive now.
	from := t0
	now := t0.Add(StripWindow)
	cov := GatewayCoverage(nil, from, now, now)
	if got := ObservedWithin(cov, from, now); got != StripWindow {
		t.Fatalf("expected full window observed, got %v", got)
	}
	// A marker older than the window means the gateway was not running at all.
	cov = GatewayCoverage(nil, from, now, from.Add(-time.Hour))
	if len(cov) != 0 {
		t.Fatalf("expected no coverage for stale marker, got %v", cov)
	}
}

func TestPeerSegmentsUsesCurrentStateWithoutTransitions(t *testing.T) {
	from := t0
	now := t0.Add(StripWindow)
	segs := PeerSegments(nil, PeerHistoryKey("home"), from, now, StateConnected)
	if len(segs) != 1 || segs[0].State != StateConnected {
		t.Fatalf("segments: %+v", segs)
	}
}

func TestStripBinsMarkGapsAndWorstState(t *testing.T) {
	from := t0
	now := from.Add(StripWindow)
	// Covered the whole window, peer offline for 30 minutes starting at 1h.
	cov := []Span{{From: from, To: now}}
	events := []history.Event{
		{Kind: KindPeer, Key: PeerHistoryKey("home"), State: StateConnected, At: from},
		{Kind: KindPeer, Key: PeerHistoryKey("home"), State: StateOffline, At: from.Add(time.Hour)},
		{Kind: KindPeer, Key: PeerHistoryKey("home"), State: StateConnected, At: from.Add(90 * time.Minute)},
	}
	segs := PeerSegments(events, PeerHistoryKey("home"), from, now, StateConnected)
	bins := StripBins(segs, cov, from, StripBinsNum, StripBin)
	if len(bins) != 96 {
		t.Fatalf("bins: %d", len(bins))
	}
	// Outage is minutes 60-90: bins 4 and 5.
	if bins[3] != StateConnected || bins[4] != StateOffline || bins[5] != StateOffline || bins[6] != StateConnected {
		t.Fatalf("unexpected bins around outage: %v", bins[3:7])
	}

	// Gateway was down for the first hour: those bins are no_data, not offline.
	covAfter := []Span{{From: from.Add(time.Hour), To: now}}
	bins = StripBins(segs, covAfter, from, StripBinsNum, StripBin)
	if bins[0] != StateNoData || bins[3] != StateNoData {
		t.Fatalf("gateway-down bins must be no_data: %v", bins[:4])
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestMonitor(t *testing.T, probe ProbeFunc, clock *fakeClock) (*Monitor, *history.Store) {
	t.Helper()
	dir := t.TempDir()
	hist := history.NewStore(filepath.Join(dir, "state-history.jsonl"))
	m := &Monitor{
		hist:       hist,
		markerPath: filepath.Join(dir, gatewayMarkerFile),
		probe:      probe,
		now:        clock.now,
		sem:        make(chan struct{}, ProbeConcurrency),
		probes:     map[string]*probeState{},
		states:     map[string]string{},
	}
	return m, hist
}

// waitProbes blocks until no probe is in flight for the peer.
func waitProbes(m *Monitor, id string) {
	for i := 0; i < 1000; i++ {
		m.mu.Lock()
		busy := m.probes[id] != nil && m.probes[id].inFlight
		m.mu.Unlock()
		if !busy {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func TestMonitorHysteresisAndTransitions(t *testing.T) {
	clock := &fakeClock{t: t0}
	fail := false
	probe := func(ctx context.Context, host string, port int) error {
		if fail {
			return errors.New("connection refused")
		}
		return nil
	}
	m, hist := newTestMonitor(t, probe, clock)
	ctx := context.Background()
	peer := PeerRecord{ID: "home", TailnetHost: "100.64.0.2", LastSeen: clock.t}

	// First evaluation: heartbeat fresh, successful probe, state connected.
	m.evaluate(ctx, clock.t, []PeerRecord{peer})
	waitProbes(m, "home")
	// One failure must not degrade the peer (hysteresis).
	fail = true
	clock.t = clock.t.Add(ProbeInterval + ProbeJitter)
	peer.LastSeen = clock.t
	m.evaluate(ctx, clock.t, []PeerRecord{peer})
	waitProbes(m, "home")
	if !m.Reachable("home") {
		t.Fatal("one failure must not degrade")
	}
	// Second consecutive failure degrades.
	clock.t = clock.t.Add(ProbeInterval + ProbeJitter)
	peer.LastSeen = clock.t
	m.evaluate(ctx, clock.t, []PeerRecord{peer})
	waitProbes(m, "home")
	if m.Reachable("home") {
		t.Fatal("two failures must degrade")
	}
	clock.t = clock.t.Add(MonitorTick)
	peer.LastSeen = clock.t
	m.evaluate(ctx, clock.t, []PeerRecord{peer})

	evs, _ := hist.Since(time.Time{})
	var states []string
	for _, e := range evs {
		if e.Kind == KindPeer {
			states = append(states, e.State)
		}
	}
	if len(states) != 2 || states[0] != StateConnected || states[1] != StateDegraded {
		t.Fatalf("transitions: %v", states)
	}

	// Recovery needs two successes.
	fail = false
	clock.t = clock.t.Add(ProbeInterval + ProbeJitter)
	peer.LastSeen = clock.t
	m.evaluate(ctx, clock.t, []PeerRecord{peer})
	waitProbes(m, "home")
	if m.Reachable("home") {
		t.Fatal("one success must not recover")
	}
	clock.t = clock.t.Add(ProbeInterval + ProbeJitter)
	peer.LastSeen = clock.t
	m.evaluate(ctx, clock.t, []PeerRecord{peer})
	waitProbes(m, "home")
	if !m.Reachable("home") {
		t.Fatal("two successes must recover")
	}
}

func TestMonitorRecordsGatewayGapAfterRestart(t *testing.T) {
	clock := &fakeClock{t: t0}
	m, hist := newTestMonitor(t, TCPProbe, clock)
	// Marker left behind by a previous run, 10 minutes ago.
	if err := writeMarker(m.markerPath, t0.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	m.recordLiveness(clock.t)
	evs, _ := hist.Since(time.Time{})
	var got []string
	for _, e := range evs {
		if e.Kind == KindGateway {
			got = append(got, e.State)
		}
	}
	if len(got) != 2 || got[0] != GatewayGap || got[1] != GatewayUp {
		t.Fatalf("gateway events: %v", got)
	}
}
