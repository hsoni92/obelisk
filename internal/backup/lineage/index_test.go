package lineage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterNodeChainsParent(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	if err := os.WriteFile(cfg, []byte("instances: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Witness{Schedule: "@daily", ReplicationFactor: 2, DataRetentionDays: 14}
	n1, err := RegisterNode(cfg, RegisterInput{
		ServiceType: "postgres",
		InstanceID:  "main",
		ObjectKey:   "data/postgres/main/a.sql",
		Size:        10,
		Trigger:     TriggerManual,
		Witness:     w,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n1.ParentID != "" {
		t.Fatalf("first parent=%q", n1.ParentID)
	}
	n2, err := RegisterNode(cfg, RegisterInput{
		ServiceType: "postgres",
		InstanceID:  "main",
		ObjectKey:   "data/postgres/main/b.sql",
		Size:        20,
		Witness:     w,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n2.ParentID != n1.ID {
		t.Fatalf("parent=%q want %q", n2.ParentID, n1.ID)
	}
	idx, err := Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	f := idx.Forests[InstanceKey("postgres", "main")]
	if f.ActiveBackupID != n2.ID {
		t.Fatalf("active=%q", f.ActiveBackupID)
	}
}

func TestCascadeDelete(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	_ = os.WriteFile(cfg, []byte("instances: []\n"), 0o600)
	w := Witness{DataRetentionDays: 7, ReplicationFactor: 1}
	_, _ = RegisterNode(cfg, RegisterInput{ServiceType: "postgres", InstanceID: "a", ObjectKey: "data/postgres/a/1.sql", Witness: w})
	n2, _ := RegisterNode(cfg, RegisterInput{ServiceType: "postgres", InstanceID: "a", ObjectKey: "data/postgres/a/2.sql", Witness: w})
	n3, _ := RegisterNode(cfg, RegisterInput{ServiceType: "postgres", InstanceID: "a", ObjectKey: "data/postgres/a/3.sql", Witness: w})
	res, err := CascadeDelete(cfg, "postgres", "a", n2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DeletedIDs) != 2 {
		t.Fatalf("deleted=%v", res.DeletedIDs)
	}
	idx, _ := Load(cfg)
	f := idx.Forests[InstanceKey("postgres", "a")]
	if f.ActiveBackupID != n3.ID {
		t.Fatalf("active=%q want %q", f.ActiveBackupID, n3.ID)
	}
	if !f.Nodes[n2.ID].Tombstone {
		t.Fatal("n2 not tombstoned")
	}
}

func TestDeleteEntireForest(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	_ = os.WriteFile(cfg, []byte("instances: []\n"), 0o600)
	w := Witness{DataRetentionDays: 7, ReplicationFactor: 1}
	_, _ = RegisterNode(cfg, RegisterInput{ServiceType: "postgres", InstanceID: "x", ObjectKey: "data/postgres/x/a.sql", Witness: w})
	_, _ = RegisterNode(cfg, RegisterInput{ServiceType: "postgres", InstanceID: "x", ObjectKey: "data/postgres/x/b.sql", Witness: w})
	res, err := DeleteEntireForest(cfg, "postgres", "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DeletedIDs) != 2 {
		t.Fatalf("deleted=%d", len(res.DeletedIDs))
	}
	idx, _ := Load(cfg)
	if _, ok := idx.Forests[InstanceKey("postgres", "x")]; ok {
		t.Fatal("forest should be removed")
	}
}

func TestPlanCloneForestDoesNotTouchSourceKey(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	if err := os.WriteFile(cfg, []byte("instances: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Witness{ReplicationFactor: 1}
	_, _ = RegisterNode(cfg, RegisterInput{
		ServiceType: "postgres",
		InstanceID:  "db",
		ObjectKey:   "data/postgres/db/1.sql",
		Size:        100,
		Witness:     w,
	})
	n2, _ := RegisterNode(cfg, RegisterInput{
		ServiceType: "postgres",
		InstanceID:  "db",
		ObjectKey:   "data/postgres/db/2.sql",
		Size:        200,
		Witness:     w,
	})
	idx, err := Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	before := len(idx.Forests)
	plan, err := PlanCloneForest(
		idx,
		"postgres",
		"db",
		CloneScopeLineage,
		"",
		"",
		func(_, cloneID, srcKey string, _ int) (string, error) {
			return "data/postgres/" + cloneID + "/copy-" + srcKey, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.TargetForest.CloneSourceKey != "postgres/db" {
		t.Fatalf("source=%q", plan.TargetForest.CloneSourceKey)
	}
	if len(plan.Jobs) != 2 {
		t.Fatalf("jobs=%d", len(plan.Jobs))
	}
	if plan.TargetForest.InstanceID == "db" {
		t.Fatal("clone must use new instance id")
	}
	idx2, _ := Load(cfg)
	if len(idx2.Forests) != before {
		t.Fatal("PlanCloneForest must not persist")
	}
	if plan.TargetForest.ActiveBackupID == "" {
		t.Fatal("expected active on clone")
	}
	if _, ok := plan.TargetForest.Nodes[n2.ID]; ok {
		t.Fatal("clone node ids must be remapped")
	}
}
