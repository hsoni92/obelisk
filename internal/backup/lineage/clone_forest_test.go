package lineage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanCloneForestSnapshotUsesActive(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	if err := os.WriteFile(cfg, []byte("instances: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Witness{ReplicationFactor: 1}
	_, _ = RegisterNode(cfg, RegisterInput{
		ServiceType: "postgres",
		InstanceID:  "x",
		ObjectKey:   "data/postgres/x/a.sql",
		Size:        1,
		Witness:     w,
	})
	active, _ := RegisterNode(cfg, RegisterInput{
		ServiceType: "postgres",
		InstanceID:  "x",
		ObjectKey:   "data/postgres/x/b.sql",
		Size:        2,
		Witness:     w,
	})
	idx, _ := Load(cfg)
	plan, err := PlanCloneForest(idx, "postgres", "x", CloneScopeSnapshot, "", "", stubObjectKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Jobs) != 1 {
		t.Fatalf("jobs=%d", len(plan.Jobs))
	}
	if plan.Jobs[0].SourceObjectKey != active.ObjectKey {
		t.Fatalf("want active object %q", active.ObjectKey)
	}
}

func stubObjectKey(_, cloneID, _ string, _ int) (string, error) {
	return "data/postgres/" + cloneID + "/snap.sql", nil
}