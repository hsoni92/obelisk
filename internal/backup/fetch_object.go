package backup

import (
	"context"
	"fmt"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// FetchObjectFromDestinations downloads a backup object from the first reachable destination.
func FetchObjectFromDestinations(
	ctx context.Context,
	configPath string,
	spec obelisk.Spec,
	objectKey string,
) ([]byte, error) {
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := LoadDestinationCredentials(configPath, d.ID)
		if err != nil || !ok {
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
		body, err := GetObject(ctx, d.Endpoint, bucket, objectKey, creds)
		if err == nil {
			return body, nil
		}
	}
	return nil, fmt.Errorf("could not download %q from any destination", objectKey)
}
