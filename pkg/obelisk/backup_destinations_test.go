package obelisk

import "testing"

func TestEffectiveDestinationsLegacyRemote(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Machine.ID = "obelisk-deadbeef"
	spec.Backup.Remote.Endpoint = "https://s3.us-east-1.amazonaws.com"
	dests := EffectiveDestinations(spec)
	if len(dests) != 1 || dests[0].ID != "primary" {
		t.Fatalf("expected legacy primary, got %+v", dests)
	}
}

func TestValidateReplicationFactor(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Machine.ID = "obelisk-deadbeef"
	spec.Backup.Policy.ReplicationFactor = 2
	spec.Backup.Destinations = []BackupDestination{
		{ID: "a", Label: "A", Endpoint: "https://x", Enabled: true},
	}
	if err := ValidateBackupPolicy(spec); err == nil {
		t.Fatal("expected RF > enabled error")
	}
	spec.Backup.Destinations = append(spec.Backup.Destinations,
		BackupDestination{ID: "b", Label: "B", Endpoint: "https://y", Enabled: true},
	)
	if err := ValidateBackupPolicy(spec); err != nil {
		t.Fatal(err)
	}
}