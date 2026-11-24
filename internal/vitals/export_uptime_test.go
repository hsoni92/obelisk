package vitals

import (
	"testing"
	"time"
)

func TestValidRunningSince(t *testing.T) {
	if _, ok := validRunningSince(time.Time{}); ok {
		t.Fatal("zero time should be invalid")
	}
	if _, ok := validRunningSince(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)); ok {
		t.Fatal("year 1 should be invalid")
	}
	now := time.Now().UTC()
	if got, ok := validRunningSince(now); !ok || !got.Equal(now) {
		t.Fatalf("now: got %v ok=%v", got, ok)
	}
}
