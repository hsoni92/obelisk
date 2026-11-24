package obelisk

import "testing"

func TestBackupScopeExcludesRedis(t *testing.T) {
	if ServiceTypeSupportsBackup("redis") {
		t.Fatal("redis must not be backed up")
	}
	for _, typ := range []string{"postgres", "mysql", "mongodb", "rustfs", "qdrant"} {
		if !ServiceTypeSupportsBackup(typ) {
			t.Fatalf("%s should support backup", typ)
		}
	}
}

func TestInstanceBackupOptOut(t *testing.T) {
	off := false
	inst := ServiceInstance{Type: "postgres", Enabled: true, Backup: InstanceBackupSettings{Enabled: &off}}
	if InstanceDataBackupEnabled(inst) {
		t.Fatal("expected opt-out")
	}
}
