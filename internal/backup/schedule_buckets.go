package backup

import (
	"fmt"
	"log"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// scheduleBuckets maps cron expression -> instances that use that effective schedule.
func scheduleBuckets(spec obelisk.Spec) (map[string]InstanceSet, error) {
	global := strings.TrimSpace(obelisk.ResolvedBackupPolicy(spec).Schedule)
	if global == "" {
		return nil, fmt.Errorf("global backup schedule is empty")
	}
	if err := ValidateCronSchedule(global); err != nil {
		return nil, fmt.Errorf("global schedule: %w", err)
	}

	buckets := map[string]InstanceSet{}
	for _, inst := range spec.Instances {
		if !inst.Enabled || !obelisk.InstanceDataBackupEnabled(inst) {
			continue
		}
		view, _ := obelisk.ResolvedInstanceBackupPolicy(inst, spec.Backup.Policy)
		sched := strings.TrimSpace(view.Schedule)
		if err := ValidateCronSchedule(sched); err != nil {
			log.Printf(
				"backup schedule: skip %s/%s — invalid cron %q: %v",
				inst.Type, inst.ID, sched, err,
			)
			continue
		}
		if buckets[sched] == nil {
			buckets[sched] = InstanceSet{}
		}
		buckets[sched].Add(inst.Type, inst.ID)
	}

	if len(buckets) == 0 {
		// No data services yet; still arm the global job so backups begin when instances appear.
		buckets[global] = InstanceSet{}
	}
	return buckets, nil
}