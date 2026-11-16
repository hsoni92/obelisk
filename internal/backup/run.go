package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// RunOutcome summarizes a manual backup run.
type RunOutcome struct {
	UploadedDestinations int
	RequiredReplication  int
	ObjectKey            string
	DataObjects          []string
	DataSkipped          []string
	Errors               []string
}

// RunNow uploads a control-plane snapshot to each enabled destination with credentials.
func (r *Runner) RunNow(ctx context.Context) (RunOutcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ready, msg, err := r.Ready()
	if err != nil {
		return RunOutcome{}, err
	}
	if !ready {
		return RunOutcome{}, fmt.Errorf("backup not configured: %s", msg)
	}
	spec, cfg, err := r.loadSpec()
	if err != nil {
		return RunOutcome{}, err
	}
	policy := obelisk.ResolvedBackupPolicy(spec)
	rf := policy.ReplicationFactor
	if rf < 1 {
		rf = 1
	}

	// Settings durability is the GitHub blueprint. S3 receives data/ only (no config/ tarball).
	type artifact struct {
		key  string
		inst string // "<type>/<id>"
		body []byte
	}
	var artifacts []artifact
	dumps := CollectDataDumps(ctx, spec, r.InstanceStates)
	out := RunOutcome{RequiredReplication: rf}
	for _, d := range dumps {
		if d.Skipped {
			out.DataSkipped = append(out.DataSkipped,
				fmt.Sprintf("%s/%s: %s", d.ServiceType, d.InstanceID, d.SkipReason))
			continue
		}
		if len(d.Body) > 0 {
			out.DataObjects = append(out.DataObjects, d.ObjectKey)
			artifacts = append(artifacts, artifact{key: d.ObjectKey, inst: d.ServiceType + "/" + d.InstanceID, body: d.Body})
		}
	}

	wantData := countEligibleDataInstances(spec, r.InstanceStates)
	if wantData > 0 && len(out.DataObjects) == 0 {
		reason := strings.Join(out.DataSkipped, "; ")
		if reason == "" {
			reason = "unknown"
		}
		return out, fmt.Errorf(
			"no database backups in S3 (look under data/<type>/<id>/): %s",
			reason,
		)
	}
	if len(artifacts) == 0 {
		_ = UpdateLastRun(cfg)
		return out, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	backedUp := map[string]bool{}
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := LoadDestinationCredentials(cfg, d.ID)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", d.Label, err))
			continue
		}
		if !ok {
			legacy, legOK, _ := LoadCredentials(cfg)
			if legOK && (d.ID == "primary" || len(spec.Backup.Destinations) == 0) {
				creds = legacy
				ok = true
			}
		}
		if !ok {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: credentials missing", d.Label))
			continue
		}
		bucket := obelisk.DestinationBucket(spec, d)
		var destErr error
		for _, a := range artifacts {
			if err := PutObject(ctx, d.Endpoint, bucket, a.key, creds, a.body); err != nil {
				destErr = err
				break
			}
			backedUp[a.inst] = true
		}
		if destErr != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", d.Label, destErr))
			continue
		}
		out.UploadedDestinations++
		_ = UpdateDestinationUpload(cfg, d.ID)
		if b, n, invErr := BucketUsage(ctx, d.Endpoint, bucket, creds, 2000); invErr == nil {
			_ = UpdateDestinationInventory(cfg, d.ID, b, n)
		}
	}

	keys := make([]string, 0, len(backedUp))
	for k := range backedUp {
		keys = append(keys, k)
	}
	_ = UpdateInstanceBackups(cfg, keys)
	_ = UpdateLastRun(cfg)

	if out.UploadedDestinations < rf {
		return out, fmt.Errorf(
			"replication not met (%d/%d uploads): %s",
			out.UploadedDestinations,
			rf,
			strings.Join(out.Errors, "; "),
		)
	}
	return out, nil
}
