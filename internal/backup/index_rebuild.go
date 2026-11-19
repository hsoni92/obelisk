package backup

import (
	"context"

	"github.com/hsoni92/obelisk/internal/backup/lineage"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// RebuildLineageIndex lists data/ on the first ready destination and backfills missing forests.
func RebuildLineageIndex(ctx context.Context, configPath string, spec obelisk.Spec) (int, error) {
	policy := obelisk.ResolvedBackupPolicy(spec)
	witness := lineage.Witness{
		Schedule:          policy.Schedule,
		ReplicationFactor: policy.ReplicationFactor,
		DataRetentionDays: policy.DataRetentionDays,
	}
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := LoadDestinationCredentials(configPath, d.ID)
		if err != nil {
			return 0, err
		}
		if !ok {
			legacy, legOK, _ := LoadCredentials(configPath)
			if legOK {
				creds = legacy
				ok = true
			}
		}
		if !ok {
			continue
		}
		bucket := obelisk.DestinationBucket(spec, d)
		keys, err := ListObjectKeys(ctx, d.Endpoint, bucket, obelisk.BackupDataPrefix, creds, 5000)
		if err != nil {
			return 0, err
		}
		return lineage.BackfillFromKeys(configPath, keys, witness)
	}
	return 0, nil
}