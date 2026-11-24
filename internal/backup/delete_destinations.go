package backup

import (
	"context"
	"fmt"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// DeleteObjectsAllDestinations removes keys from every enabled destination with credentials.
func DeleteObjectsAllDestinations(ctx context.Context, configPath string, spec obelisk.Spec, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := LoadDestinationCredentials(configPath, d.ID)
		if err != nil {
			return err
		}
		if !ok {
			legacy, legOK, _ := LoadCredentials(configPath)
			if legOK && (d.ID == "primary" || len(spec.Backup.Destinations) == 0) {
				creds = legacy
				ok = true
			}
		}
		if !ok {
			continue
		}
		bucket := obelisk.DestinationBucket(spec, d)
		for _, key := range keys {
			if key == "" {
				continue
			}
			if err := DeleteObject(ctx, d.Endpoint, bucket, key, creds); err != nil {
				return fmt.Errorf("%s: %w", d.Label, err)
			}
		}
	}
	return nil
}
