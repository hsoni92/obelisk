package obelisk

import "testing"

func TestEnsureMachineIdentityAssignsBucketName(t *testing.T) {
	spec := DefaultSpec("local")
	if !EnsureMachineIdentity(&spec) {
		t.Fatal("expected new machine id")
	}
	if err := ValidateMachineID(spec.Machine.ID); err != nil {
		t.Fatal(err)
	}
	if spec.BackupBucketName() != spec.Machine.ID {
		t.Fatalf("bucket name = machine id, got %q", spec.BackupBucketName())
	}
	if spec.BackupRemoteReady() {
		t.Fatal("endpoint still required")
	}
	spec.Backup.Remote.Endpoint = "https://example.com"
	if !spec.BackupRemoteReady() {
		t.Fatal("expected ready with endpoint and id")
	}
}
