package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/backup/lineage"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Deploy-from-backup (new service) — not wired to POST /backups/clone; see CloneBackup.

// CloneOutcome reports deploy-from-backup results.
type CloneOutcome struct {
	Instance     obelisk.ServiceInstance
	RestoreOK    bool
	RestoreError string
}

// WaitInstanceRunning polls until the instance state is running or timeout.
func WaitInstanceRunning(
	ctx context.Context,
	rec controller.Reconciler,
	spec obelisk.Spec,
	serviceType, instanceID string,
	timeout time.Duration,
) error {
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	key := serviceType + "/" + instanceID
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s to be running", key)
		}
		states := rec.ObserveServiceStates(ctx, spec)
		if states[key] == "running" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// PrepareCloneInstance validates the backup and appends a new instance to spec (caller must Save).
func PrepareCloneInstance(
	configPath string,
	spec *obelisk.Spec,
	sourceType, sourceInstanceID, backupID, newID, displayName string,
) (obelisk.ServiceInstance, lineage.Node, error) {
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	if !obelisk.ServiceTypeSupportsBackup(sourceType) {
		return obelisk.ServiceInstance{}, lineage.Node{}, fmt.Errorf("type %q does not support data backup", sourceType)
	}
	idx, err := lineage.Load(configPath)
	if err != nil {
		return obelisk.ServiceInstance{}, lineage.Node{}, err
	}
	backupID = strings.TrimSpace(backupID)
	if backupID == "" {
		_, forest, ok := lineage.ListNodes(idx, sourceType, sourceInstanceID)
		if !ok || strings.TrimSpace(forest.ActiveBackupID) == "" {
			return obelisk.ServiceInstance{}, lineage.Node{}, fmt.Errorf("no active backup on source instance")
		}
		backupID = forest.ActiveBackupID
	}
	node, _, ok := lineage.NodeByID(idx, sourceType, sourceInstanceID, backupID)
	if !ok {
		return obelisk.ServiceInstance{}, lineage.Node{}, fmt.Errorf("backup %q not found", backupID)
	}
	if newID == "" {
		newID = obelisk.AllocateInstanceIDForType(spec, sourceType)
	}
	inst, err := obelisk.NewServiceInstance(sourceType, newID, displayName)
	if err != nil {
		return obelisk.ServiceInstance{}, lineage.Node{}, err
	}
	if _, _, taken := obelisk.FindInstanceByTypeAndID(spec, inst.Type, inst.ID); taken {
		return obelisk.ServiceInstance{}, lineage.Node{}, fmt.Errorf("instance id %q already exists", inst.ID)
	}
	if node.Witness.ReplicationFactor > 0 {
		rf := node.Witness.ReplicationFactor
		inst.Backup.PolicyOverride = &obelisk.InstanceBackupPolicyOverride{
			ReplicationFactor: &rf,
		}
	}
	spec.Instances = append(spec.Instances, inst)
	return inst, node, nil
}

// CompleteCloneRestore downloads the backup, waits for the new instance, and restores postgres data.
func CompleteCloneRestore(
	ctx context.Context,
	configPath, helmRoot string,
	spec obelisk.Spec,
	inst obelisk.ServiceInstance,
	node lineage.Node,
	sourceBackupID string,
) (CloneOutcome, error) {
	out := CloneOutcome{Instance: inst}
	rec := controller.Reconciler{ConfigPath: configPath, HelmRoot: helmRoot}
	controller.ReconcileAsync(rec)

	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()

	dump, err := FetchObjectFromDestinations(ctx, configPath, spec, node.ObjectKey)
	if err != nil {
		return out, err
	}
	if err := WaitInstanceRunning(ctx, rec, spec, inst.Type, inst.ID, 3*time.Minute); err != nil {
		out.RestoreError = err.Error()
		_ = lineage.SetForestProvenance(configPath, inst.Type, inst.ID, sourceBackupID)
		return out, fmt.Errorf("instance created; restore pending: %w", err)
	}
	if err := RestoreInstanceData(ctx, inst, dump); err != nil {
		out.RestoreError = err.Error()
		_ = lineage.SetForestProvenance(configPath, inst.Type, inst.ID, sourceBackupID)
		return out, fmt.Errorf("instance created; restore failed: %w", err)
	}
	out.RestoreOK = true
	if err := lineage.SetForestProvenance(configPath, inst.Type, inst.ID, sourceBackupID); err != nil {
		return out, err
	}
	return out, nil
}
