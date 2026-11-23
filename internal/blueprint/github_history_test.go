package blueprint

import (
	"testing"
	"time"
)

func TestSelectRetentionParent_newDayAppend(t *testing.T) {
	tipDay := utcDay(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
	newDay := utcDay(time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC))
	ancestors := []string{"c0", "c1", "c2"}
	parent := selectRetentionParent(ancestors, tipDay, newDay)
	if parent != "c0" {
		t.Fatalf("parent = %q, want c0", parent)
	}
}

func TestSelectRetentionParent_sameDayReplace(t *testing.T) {
	day := utcDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	ancestors := []string{"c0", "c1", "c2"}
	parent := selectRetentionParent(ancestors, day, day)
	if parent != "c1" {
		t.Fatalf("parent = %q, want c1", parent)
	}
}

func TestSelectRetentionParent_pruneDepth(t *testing.T) {
	day := utcDay(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	newDay := utcDay(time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC))
	ancestors := make([]string, 30)
	for i := range ancestors {
		ancestors[i] = "c" + string(rune('0'+i))
	}
	parent := selectRetentionParent(ancestors, day, newDay)
	if parent != "c1" {
		t.Fatalf("parent = %q, want c1", parent)
	}
}