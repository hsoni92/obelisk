package backup

import (
	"context"
	"fmt"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// InstanceDataPrefix is the object-store prefix for one service instance's backups.
func InstanceDataPrefix(serviceType, instanceID string) string {
	return fmt.Sprintf("%s%s/%s/", obelisk.BackupDataPrefix, strings.ToLower(strings.TrimSpace(serviceType)), strings.TrimSpace(instanceID))
}

// ListInstanceBackupObjectKeys lists keys under data/<type>/<id>/ on the first reachable destination.
func ListInstanceBackupObjectKeys(ctx context.Context, configPath string, spec obelisk.Spec, serviceType, instanceID string) ([]string, error) {
	prefix := InstanceDataPrefix(serviceType, instanceID)
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := LoadDestinationCredentials(configPath, d.ID)
		if err != nil {
			return nil, err
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
		return ListObjectKeys(ctx, d.Endpoint, bucket, prefix, creds, 10000)
	}
	return nil, nil
}

// PurgeInstanceBackupPrefix deletes every object under data/<type>/<id>/ from all destinations.
func PurgeInstanceBackupPrefix(ctx context.Context, configPath string, spec obelisk.Spec, serviceType, instanceID string) ([]string, error) {
	keys, err := ListInstanceBackupObjectKeys(ctx, configPath, spec, serviceType, instanceID)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}
	if err := DeleteObjectsAllDestinations(ctx, configPath, spec, keys); err != nil {
		return keys, err
	}
	return keys, nil
}