package mailgateway

import "testing"

func TestRotateListAdvances(t *testing.T) {
	s := NewSelector([]mailAccount{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	})
	first := s.RotateList()
	if len(first) != 3 || first[0].ID == "" {
		t.Fatalf("unexpected first rotation: %+v", first)
	}
	second := s.RotateList()
	if second[0].ID == first[0].ID {
		t.Fatalf("expected rotation advance, first=%s second=%s", first[0].ID, second[0].ID)
	}
}