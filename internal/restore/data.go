package restore

import (
	"context"
	"path/filepath"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// RestoreDataFiles downloads data/ objects for each enabled destination.
// A missing credential or empty prefix is not an error; the count is how many
// objects were written. config/ is never fetched.
func RestoreDataFiles(ctx context.Context, configPath string, spec obelisk.Spec) (int, error) {
	destDir := filepath.Join(config.CredentialsDir(configPath), "restore-data", spec.BackupBucketName())
	total := 0
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := backup.LoadDestinationCredentials(configPath, d.ID)
		if err != nil {
			return total, err
		}
		if !ok {
			legacy, legOK, err := backup.LoadCredentials(configPath)
			if err != nil {
				return total, err
			}
			if legOK && (d.ID == "primary" || len(spec.Backup.Destinations) == 0) {
				creds = legacy
				ok = true
			}
		}
		if !ok {
			continue
		}
		bucket := obelisk.DestinationBucket(spec, d)
		n, err := backup.DownloadPrefix(ctx, d.Endpoint, bucket, obelisk.BackupDataPrefix, creds, destDir)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
