package obelisk

import (
	"fmt"
	"strings"
)

// Data backup scope: postgres, mysql, mongodb, rustfs, qdrant (not redis, searxng, redpanda, unimeter, bunqueue, …).
var BackupDataServiceTypes = []string{"postgres", "mysql", "mongodb", "rustfs", "qdrant"}

// InstanceBackupPolicyOverride overrides global backup.policy for one instance.
type InstanceBackupPolicyOverride struct {
	Schedule          *string `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	ReplicationFactor *int    `yaml:"replication_factor,omitempty" json:"replication_factor,omitempty"`
	DataRetentionDays *int    `yaml:"data_retention_days,omitempty" json:"data_retention_days,omitempty"`
}

// InstanceBackupSettings is per-instance DR participation (stored in obelisk.yaml).
type InstanceBackupSettings struct {
	Enabled        *bool                         `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	PolicyOverride *InstanceBackupPolicyOverride `yaml:"policy_override,omitempty" json:"policy_override,omitempty"`
}

// InstanceBackupPolicyView is effective policy shown in the UI.
type InstanceBackupPolicyView struct {
	Schedule          string `json:"schedule"`
	ReplicationFactor int    `json:"replication_factor"`
	DataRetentionDays int    `json:"data_retention_days"`
	Overridden        bool   `json:"overridden"`
}

// InstanceBackupView is API metadata for Services / DR UI.
type InstanceBackupView struct {
	Supported bool                     `json:"supported"`
	Enabled   bool                     `json:"enabled"`
	Method    string                   `json:"method,omitempty"`
	Prefix    string                   `json:"object_prefix,omitempty"`
	Policy    InstanceBackupPolicyView `json:"policy"`
}

// ServiceTypeSupportsBackup reports whether this service type can be in data/ backups.
func ServiceTypeSupportsBackup(typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	for _, t := range BackupDataServiceTypes {
		if t == typ {
			return true
		}
	}
	return false
}

// BackupMethodForType describes how v1 backs up a supported type.
func BackupMethodForType(typ string) string {
	switch strings.ToLower(typ) {
	case "postgres":
		return "pg_dump"
	case "mysql":
		return "mysqldump"
	case "mongodb":
		return "mongodump"
	case "rustfs":
		return "object-sync"
	case "qdrant":
		return "snapshot"
	default:
		return ""
	}
}

// ResolvedInstanceBackupPolicy merges global defaults with per-instance overrides.
func ResolvedInstanceBackupPolicy(inst ServiceInstance, global BackupPolicy) (InstanceBackupPolicyView, bool) {
	g := ResolvedBackupPolicy(Spec{Backup: Backup{Policy: global}})
	view := InstanceBackupPolicyView{
		Schedule:          g.Schedule,
		ReplicationFactor: g.ReplicationFactor,
		DataRetentionDays: g.DataRetentionDays,
	}
	overridden := false
	if inst.Backup.PolicyOverride == nil {
		return view, false
	}
	po := inst.Backup.PolicyOverride
	if po.Schedule != nil && strings.TrimSpace(*po.Schedule) != "" {
		view.Schedule = strings.TrimSpace(*po.Schedule)
		overridden = true
	}
	if po.ReplicationFactor != nil && *po.ReplicationFactor > 0 {
		view.ReplicationFactor = *po.ReplicationFactor
		overridden = true
	}
	if po.DataRetentionDays != nil && *po.DataRetentionDays > 0 {
		view.DataRetentionDays = *po.DataRetentionDays
		overridden = true
	}
	view.Overridden = overridden
	return view, overridden
}

// ValidateInstanceBackupPolicyOverride checks instance RF against cluster destinations.
func ValidateInstanceBackupPolicyOverride(spec Spec, inst ServiceInstance) error {
	if inst.Backup.PolicyOverride == nil || inst.Backup.PolicyOverride.ReplicationFactor == nil {
		return nil
	}
	rf := *inst.Backup.PolicyOverride.ReplicationFactor
	if rf < 1 {
		return fmt.Errorf("instance %s: replication_factor must be >= 1", inst.ID)
	}
	enabled := len(EnabledBackupDestinations(spec))
	if enabled > 0 && rf > enabled {
		return fmt.Errorf(
			"instance %s: replication_factor (%d) exceeds enabled destinations (%d)",
			inst.ID, rf, enabled,
		)
	}
	return nil
}

// BuildInstanceBackupView returns deployment-linked backup metadata.
func BuildInstanceBackupView(inst ServiceInstance, global BackupPolicy) InstanceBackupView {
	supported := ServiceTypeSupportsBackup(inst.Type)
	enabled := supported && InstanceDataBackupEnabled(inst)
	prefix := ""
	if supported {
		prefix = BackupDataPrefix + inst.Type + "/" + inst.ID + "/"
	}
	policy, _ := ResolvedInstanceBackupPolicy(inst, global)
	return InstanceBackupView{
		Supported: supported,
		Enabled:   enabled,
		Method:    BackupMethodForType(inst.Type),
		Prefix:    prefix,
		Policy:    policy,
	}
}

// InstanceDataBackupEnabled is true when this instance opts in to data backups.
func InstanceDataBackupEnabled(inst ServiceInstance) bool {
	if !ServiceTypeSupportsBackup(inst.Type) {
		return false
	}
	if inst.Backup.Enabled != nil && !*inst.Backup.Enabled {
		return false
	}
	return true
}

// DefaultInstanceBackupSettings for new instances.
func DefaultInstanceBackupSettings(typ string) InstanceBackupSettings {
	if !ServiceTypeSupportsBackup(typ) {
		return InstanceBackupSettings{}
	}
	on := true
	return InstanceBackupSettings{Enabled: &on}
}