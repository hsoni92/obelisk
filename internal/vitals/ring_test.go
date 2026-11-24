package vitals

import (
	"testing"
	"time"
)

func TestRingAppendAndOrder(t *testing.T) {
	var r Ring
	now := time.Now()
	for i := 0; i < RingSize+5; i++ {
		r.Append(Slot{At: now.Add(time.Duration(i) * time.Minute), MemoryUsed: int64(i)})
	}
	if r.Count() != RingSize {
		t.Fatalf("count=%d want %d", r.Count(), RingSize)
	}
	ordered := r.Ordered()
	if len(ordered) != RingSize {
		t.Fatalf("ordered len=%d", len(ordered))
	}
	if ordered[0].MemoryUsed != 5 {
		t.Fatalf("oldest=%d want 5", ordered[0].MemoryUsed)
	}
	if ordered[len(ordered)-1].MemoryUsed != int64(RingSize+4) {
		t.Fatalf("newest=%d", ordered[len(ordered)-1].MemoryUsed)
	}
}
