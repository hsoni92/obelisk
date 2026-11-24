package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/backup/lineage"
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
	RegisteredBackups    []string
}

type runArtifact struct {
	key          string
	manifestKey  string
	manifestBody []byte
	inst         string
	serviceType  string
	instanceID   string
	body         []byte
	plannedNode  lineage.Node
}

// RunNow uploads data dumps for all eligible instances (manual trigger).
func (r *Runner) RunNow(ctx context.Context) (RunOutcome, error) {
	return r.runScoped(ctx, InstanceFilter{}, lineage.TriggerManual, nil)
}

// RunScheduledFor runs scheduled backups for only the instances in only (empty = all).
func (r *Runner) RunScheduledFor(ctx context.Context, only InstanceSet) (RunOutcome, error) {
	return r.runScoped(ctx, InstanceFilter{}, lineage.TriggerScheduled, only)
}

func (r *Runner) runScoped(ctx context.Context, filter InstanceFilter, trigger string, only InstanceSet) (RunOutcome, error) {
	if trigger == "" {
		trigger = lineage.TriggerManual
	}
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
	var dumps []DataDump
	if filter.active() {
		dumps = CollectDataDumpsForInstance(ctx, spec, r.InstanceStates, filter.ServiceType, filter.InstanceID)
	} else {
		dumps = CollectDataDumps(ctx, spec, r.InstanceStates)
		if len(only) > 0 {
			dumps = filterDumpsBySet(dumps, only)
		}
	}

	rf := requiredReplication(spec, filter, dumps)
	policy := obelisk.ResolvedBackupPolicy(spec)

	out := RunOutcome{RequiredReplication: rf}
	var artifacts []runArtifact
	machineID := spec.BackupBucketName()

	for _, d := range dumps {
		if d.Skipped {
			out.DataSkipped = append(out.DataSkipped,
				fmt.Sprintf("%s/%s: %s", d.ServiceType, d.InstanceID, d.SkipReason))
			continue
		}
		if len(d.Body) == 0 {
			continue
		}
		var inst obelisk.ServiceInstance
		for _, candidate := range spec.Instances {
			if candidate.Type == d.ServiceType && candidate.ID == d.InstanceID {
				inst = candidate
				break
			}
		}
		witness := lineage.WitnessFromSpec(spec, inst)
		planned, forest, err := lineage.PlanNode(cfg, lineage.RegisterInput{
			ServiceType: d.ServiceType,
			InstanceID:  d.InstanceID,
			ObjectKey:   d.ObjectKey,
			Size:        int64(len(d.Body)),
			Trigger:     trigger,
			Witness:     witness,
		})
		if err != nil {
			return out, err
		}
		manifest := lineage.NewManifest(planned, forest, machineID)
		manifestBody, err := manifest.JSON()
		if err != nil {
			return out, err
		}
		out.DataObjects = append(out.DataObjects, d.ObjectKey)
		artifacts = append(artifacts, runArtifact{
			key:          d.ObjectKey,
			manifestKey:  planned.ManifestKey,
			manifestBody: manifestBody,
			inst:         d.ServiceType + "/" + d.InstanceID,
			serviceType:  d.ServiceType,
			instanceID:   d.InstanceID,
			body:         d.Body,
			plannedNode:  planned,
		})
	}

	wantData := countEligibleDataInstancesInSet(spec, r.InstanceStates, only)
	if filter.active() {
		wantData = 0
		for _, inst := range spec.Instances {
			if inst.Type == filter.ServiceType && inst.ID == filter.InstanceID &&
				inst.Enabled && obelisk.InstanceDataBackupEnabled(inst) {
				wantData = 1
				break
			}
		}
	}
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
		if err := EnsureDestinationBucket(ctx, d.Endpoint, bucket, creds); err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", d.Label, err))
			continue
		}
		var destErr error
		for _, a := range artifacts {
			if err := PutObject(ctx, d.Endpoint, bucket, a.key, creds, a.body); err != nil {
				destErr = err
				break
			}
			if len(a.manifestBody) > 0 {
				if err := PutObject(ctx, d.Endpoint, bucket, a.manifestKey, creds, a.manifestBody); err != nil {
					destErr = err
					break
				}
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

	if out.UploadedDestinations < rf {
		return out, fmt.Errorf(
			"replication not met (%d/%d uploads): %s",
			out.UploadedDestinations,
			rf,
			strings.Join(out.Errors, "; "),
		)
	}

	for _, a := range artifacts {
		if err := lineage.CommitNode(cfg, a.serviceType, a.instanceID, a.plannedNode); err != nil {
			return out, err
		}
		out.RegisteredBackups = append(out.RegisteredBackups, a.plannedNode.ID)
	}
	if len(artifacts) > 0 {
		retDays := policy.DataRetentionDays
		if filter.active() {
			for _, inst := range spec.Instances {
				if inst.Type == filter.ServiceType && inst.ID == filter.InstanceID {
					view, _ := obelisk.ResolvedInstanceBackupPolicy(inst, spec.Backup.Policy)
					retDays = view.DataRetentionDays
					break
				}
			}
		}
		_, _ = lineage.PruneRetention(cfg, retDays, false)
	}

	keys := make([]string, 0, len(backedUp))
	for k := range backedUp {
		keys = append(keys, k)
	}
	_ = UpdateInstanceBackups(cfg, keys)
	_ = UpdateLastRun(cfg)

	return out, nil
}

// requiredReplication is the minimum destination copy count for this run
// (per-instance override when scoped; else max effective RF across backed-up instances).
func requiredReplication(spec obelisk.Spec, filter InstanceFilter, dumps []DataDump) int {
	global := obelisk.ResolvedBackupPolicy(spec).ReplicationFactor
	if global < 1 {
		global = 1
	}
	if filter.active() {
		for _, inst := range spec.Instances {
			if inst.Type != filter.ServiceType || inst.ID != filter.InstanceID {
				continue
			}
			view, _ := obelisk.ResolvedInstanceBackupPolicy(inst, spec.Backup.Policy)
			rf := view.ReplicationFactor
			if rf < 1 {
				rf = 1
			}
			return rf
		}
		return global
	}
	maxRF := global
	for _, d := range dumps {
		if d.Skipped || len(d.Body) == 0 {
			continue
		}
		for _, inst := range spec.Instances {
			if inst.Type != d.ServiceType || inst.ID != d.InstanceID {
				continue
			}
			view, _ := obelisk.ResolvedInstanceBackupPolicy(inst, spec.Backup.Policy)
			rf := view.ReplicationFactor
			if rf < 1 {
				rf = 1
			}
			if rf > maxRF {
				maxRF = rf
			}
			break
		}
	}
	return maxRF
}
