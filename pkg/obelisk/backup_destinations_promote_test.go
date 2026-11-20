package obelisk

import "testing"

func TestPromoteBackupDestination_SwapPrimaryID(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Machine.ID = "obelisk-deadbeef"
	spec.Backup.Destinations = []BackupDestination{
		{ID: "primary", Label: "Primary", Endpoint: "https://a.example", Enabled: true},
		{ID: "bd-2", Label: "R2", Endpoint: "https://b.example", Enabled: true},
	}
	former, promotedOld, err := PromoteBackupDestination(&spec, "bd-2")
	if err != nil {
		t.Fatal(err)
	}
	if former != "bd-2" || promotedOld != "bd-2" {
		t.Fatalf("former=%q promotedOld=%q", former, promotedOld)
	}
	if spec.Backup.Destinations[0].ID != "primary" || spec.Backup.Destinations[0].Endpoint != "https://b.example" {
		t.Fatalf("promoted slot: %+v", spec.Backup.Destinations[0])
	}
	if spec.Backup.Destinations[1].ID != "bd-2" || spec.Backup.Destinations[1].Endpoint != "https://a.example" {
		t.Fatalf("demoted: %+v", spec.Backup.Destinations[1])
	}
	if spec.Backup.Remote.Endpoint != "https://b.example" {
		t.Fatalf("remote endpoint=%q", spec.Backup.Remote.Endpoint)
	}
}

func TestIsPrimaryBackupDestination(t *testing.T) {
	spec := DefaultSpec("vps")
	spec.Backup.Destinations = []BackupDestination{
		{ID: "primary", Endpoint: "https://a", Enabled: true},
		{ID: "bd-2", Endpoint: "https://b", Enabled: true},
	}
	if !IsPrimaryBackupDestination(spec, "primary") {
		t.Fatal("primary should be primary")
	}
	if IsPrimaryBackupDestination(spec, "bd-2") {
		t.Fatal("bd-2 should not be primary")
	}
}