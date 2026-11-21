package backup

import (
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestValidateCronSchedule(t *testing.T) {
	if err := ValidateCronSchedule("0 */6 * * *"); err != nil {
		t.Fatalf("expected valid 6h cron: %v", err)
	}
	if err := ValidateCronSchedule("not a cron"); err == nil {
		t.Fatal("expected invalid cron")
	}
}

func TestScheduleBucketsPerInstanceOverride(t *testing.T) {
	global := obelisk.BackupPolicy{Schedule: "0 */6 * * *"}
	on := true
	spec := obelisk.Spec{
		Instances: []obelisk.ServiceInstance{
			{
				Type: "postgres", ID: "a", Enabled: true,
				Backup: obelisk.InstanceBackupSettings{Enabled: &on},
			},
			{
				Type: "mongodb", ID: "b", Enabled: true,
				Backup: obelisk.InstanceBackupSettings{
					Enabled: &on,
					PolicyOverride: &obelisk.InstanceBackupPolicyOverride{
						Schedule: strPtr("0 3 * * *"),
					},
				},
			},
		},
		Backup: obelisk.Backup{Policy: global},
	}
	buckets, err := scheduleBuckets(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets["0 */6 * * *"]) != 1 || !buckets["0 */6 * * *"].Contains("postgres", "a") {
		t.Fatalf("postgres should be on global schedule: %#v", buckets)
	}
	if len(buckets["0 3 * * *"]) != 1 || !buckets["0 3 * * *"].Contains("mongodb", "b") {
		t.Fatalf("mongodb should be on daily override: %#v", buckets)
	}
}

func strPtr(s string) *string { return &s }

func TestImplementationStatusWithoutRunner(t *testing.T) {
	globalScheduler.Store(nil)
	if got := ImplementationStatus(); got != "stub" {
		t.Fatalf("got %q want stub", got)
	}
}
