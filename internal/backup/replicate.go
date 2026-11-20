package backup

import (
	"context"
	"fmt"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// ObjectPair is one data object plus optional manifest sidecar.
type ObjectPair struct {
	Key          string
	Body         []byte
	ManifestKey  string
	ManifestBody []byte
}

// ReplicateObjects uploads pairs to enabled destinations until RF is met.
func ReplicateObjects(
	ctx context.Context,
	configPath string,
	spec obelisk.Spec,
	pairs []ObjectPair,
) (uploadedDestinations int, err error) {
	policy := obelisk.ResolvedBackupPolicy(spec)
	rf := policy.ReplicationFactor
	if rf < 1 {
		rf = 1
	}
	var errs []string
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, credErr := LoadDestinationCredentials(configPath, d.ID)
		if credErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", d.Label, credErr))
			continue
		}
		if !ok {
			legacy, legOK, _ := LoadCredentials(configPath)
			if legOK && (d.ID == "primary" || len(spec.Backup.Destinations) == 0) {
				creds = legacy
				ok = true
			}
		}
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: credentials missing", d.Label))
			continue
		}
		bucket := obelisk.DestinationBucket(spec, d)
		if err := EnsureDestinationBucket(ctx, d.Endpoint, bucket, creds); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", d.Label, err))
			continue
		}
		var destErr error
		for _, p := range pairs {
			if err := PutObject(ctx, d.Endpoint, bucket, p.Key, creds, p.Body); err != nil {
				destErr = err
				break
			}
			if p.ManifestKey != "" && len(p.ManifestBody) > 0 {
				if err := PutObject(ctx, d.Endpoint, bucket, p.ManifestKey, creds, p.ManifestBody); err != nil {
					destErr = err
					break
				}
			}
		}
		if destErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", d.Label, destErr))
			continue
		}
		uploadedDestinations++
		_ = UpdateDestinationUpload(configPath, d.ID)
	}
	if uploadedDestinations < rf {
		return uploadedDestinations, fmt.Errorf(
			"replication not met (%d/%d uploads): %s",
			uploadedDestinations,
			rf,
			strings.Join(errs, "; "),
		)
	}
	return uploadedDestinations, nil
}