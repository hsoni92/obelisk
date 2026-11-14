package openroutergateway

import "testing"

func TestSelectorSkipsRoundRobinOrder(t *testing.T) {
	s := NewSelector([]upstreamAccount{
		{ID: "a", Key: "1"},
		{ID: "b", Key: "2"},
	})
	ids := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		acc, ok := s.Next()
		if !ok {
			t.Fatal("expected account")
		}
		ids = append(ids, acc.ID)
	}
	want := []string{"a", "b", "a", "b"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("rr[%d]=%s want %s", i, ids[i], want[i])
		}
	}
}

func TestSelectorEmpty(t *testing.T) {
	s := NewSelector(nil)
	if _, ok := s.Next(); ok {
		t.Fatal("expected empty")
	}
}