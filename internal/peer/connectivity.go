package peer

import (
	"sort"
	"time"

	"github.com/hsoni92/obelisk/internal/history"
)

// Peer connectivity states shown on the Connectivity page.
const (
	StateConnected = "connected"
	StateDegraded  = "degraded"
	StateOffline   = "offline"
	StateNever     = "never"
	StateNoData    = "no_data"
	StateUnknown   = "unknown"
)

// History kinds and gateway event states.
const (
	KindPeer     = "peer"
	KindGateway  = "gateway"
	GatewayUp    = "up"
	GatewayGap   = "gap"
	StripWindow  = 24 * time.Hour
	StripBin     = 15 * time.Minute
	StripBinsNum = int(StripWindow / StripBin)
)

// PeerHistoryKey is the history key for one peer's state transitions.
func PeerHistoryKey(peerID string) string {
	return "peer/" + peerID
}

// CombinedState derives the one state shown for a peer from its two signals.
//
//   - heartbeat (control plane): the peer advertised within HeartbeatStaleAfter
//   - reachable (data plane): the gateway's TCP probe is passing (after hysteresis)
func CombinedState(lastSeen, now time.Time, reachable bool) string {
	if lastSeen.IsZero() {
		return StateNever
	}
	if !heartbeatFresh(lastSeen, now, HeartbeatStaleAfter) {
		return StateOffline
	}
	if !reachable {
		return StateDegraded
	}
	return StateConnected
}

// Span is a half-open time range [From, To).
type Span struct {
	From time.Time
	To   time.Time
}

// Segment is a span during which a peer held one state.
type Segment struct {
	From  time.Time
	To    time.Time
	State string
}

func clampSpan(from, to, lo, hi time.Time) (Span, bool) {
	if from.Before(lo) {
		from = lo
	}
	if to.After(hi) {
		to = hi
	}
	if !to.After(from) {
		return Span{}, false
	}
	return Span{From: from, To: to}, true
}

// GatewayCoverage returns the spans in [from, now) during which this gateway
// was observing its peers. Time outside these spans is "no data", never "down".
//
// Rules, applied to gateway events in time order:
//   - an "up" event opens an observed span;
//   - a "gap" event closes the open span at its start (the gateway was not
//     observing between At and Until);
//   - an open span at the end runs until lastAlive (the last liveness marker),
//     or to now if the marker is missing.
//
// Pruning removes events older than the retention window, so when the window
// holds no "up" event, observation is assumed from the window start, but only
// if the marker shows the gateway alive recently. Otherwise nothing is observed.
func GatewayCoverage(events []history.Event, from, now, lastAlive time.Time) []Span {
	var gw []history.Event
	for _, e := range events {
		if e.Kind == KindGateway {
			gw = append(gw, e)
		}
	}
	sort.SliceStable(gw, func(i, j int) bool { return gw[i].At.Before(gw[j].At) })

	var out []Span
	var open time.Time
	isOpen := false
	switch {
	case len(gw) == 0:
		if !lastAlive.IsZero() && !lastAlive.Before(from) {
			open, isOpen = from, true
		}
	case gw[0].State != GatewayUp:
		open, isOpen = from, true
	}

	for _, e := range gw {
		switch e.State {
		case GatewayUp:
			if !isOpen {
				open, isOpen = e.At, true
			}
		case GatewayGap:
			if isOpen {
				if s, ok := clampSpan(open, e.At, from, now); ok {
					out = append(out, s)
				}
				isOpen = false
			}
		}
	}
	if isOpen {
		end := now
		if !lastAlive.IsZero() && lastAlive.Before(now) {
			end = lastAlive
		}
		if s, ok := clampSpan(open, end, from, now); ok {
			out = append(out, s)
		}
	}
	return out
}

// PeerSegments builds one peer's state timeline over [from, now).
//
// Before the first in-window transition the state is unknown, unless no
// transition exists at all. Then the current state holds for the whole window:
// the peer has not changed state since before the window, and coverage masks
// any period the gateway did not observe.
func PeerSegments(events []history.Event, key string, from, now time.Time, current string) []Segment {
	var evs []history.Event
	for _, e := range events {
		if e.Kind == KindPeer && e.Key == key {
			evs = append(evs, e)
		}
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].At.Before(evs[j].At) })

	var segs []Segment
	var start string
	hasStart := false
	idx := 0
	for idx < len(evs) && !evs[idx].At.After(from) {
		start = evs[idx].State
		hasStart = true
		idx++
	}
	cursor := from
	if !hasStart {
		if idx >= len(evs) {
			return []Segment{{From: from, To: now, State: current}}
		}
		start = StateUnknown
	}
	segs = append(segs, Segment{From: cursor, To: nextAt(evs, idx, now), State: start})
	for ; idx < len(evs); idx++ {
		e := evs[idx]
		if !e.At.Before(now) {
			break
		}
		segs = append(segs, Segment{From: e.At, To: nextAt(evs, idx+1, now), State: e.State})
	}
	return segs
}

func nextAt(evs []history.Event, idx int, now time.Time) time.Time {
	if idx < len(evs) && evs[idx].At.Before(now) {
		return evs[idx].At
	}
	return now
}

func stateRank(s string) int {
	switch s {
	case StateOffline:
		return 0
	case StateDegraded:
		return 1
	case StateConnected:
		return 2
	}
	return -1
}

// StripBins returns one state per bin over the window starting at from.
// A bin with no gateway coverage is no_data. Otherwise it shows the worst known
// peer state inside it, so a short outage is not hidden by a longer healthy part.
func StripBins(segs []Segment, coverage []Span, from time.Time, bins int, binSize time.Duration) []string {
	out := make([]string, bins)
	for i := 0; i < bins; i++ {
		b0 := from.Add(time.Duration(i) * binSize)
		b1 := b0.Add(binSize)
		if !anyOverlap(coverage, b0, b1) {
			out[i] = StateNoData
			continue
		}
		worst := ""
		for _, s := range segs {
			if !s.From.Before(b1) || !s.To.After(b0) {
				continue
			}
			r := stateRank(s.State)
			if r < 0 {
				continue
			}
			if worst == "" || r < stateRank(worst) {
				worst = s.State
			}
		}
		if worst == "" {
			worst = StateNoData
		}
		out[i] = worst
	}
	return out
}

func anyOverlap(spans []Span, from, to time.Time) bool {
	for _, s := range spans {
		if s.From.Before(to) && s.To.After(from) {
			return true
		}
	}
	return false
}

// ObservedWithin returns how long the gateway observed peers within [from, to).
func ObservedWithin(coverage []Span, from, to time.Time) time.Duration {
	var total time.Duration
	for _, s := range coverage {
		lo, hi := s.From, s.To
		if lo.Before(from) {
			lo = from
		}
		if hi.After(to) {
			hi = to
		}
		if hi.After(lo) {
			total += hi.Sub(lo)
		}
	}
	return total
}
