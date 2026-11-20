package obelisk

import (
	"fmt"
	"strings"
)

// BackupPolicy governs default backup schedule and replication across destinations.
type BackupPolicy struct {
	Schedule          string `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	ReplicationFactor int    `yaml:"replication_factor,omitempty" json:"replication_factor,omitempty"`
	DataRetentionDays int    `yaml:"data_retention_days,omitempty" json:"data_retention_days,omitempty"`
	ConfigKeepCount   int    `yaml:"config_keep_count,omitempty" json:"config_keep_count,omitempty"`
	IncludeConfig     *bool  `yaml:"include_config,omitempty" json:"include_config,omitempty"`
}

// BackupDestination is one S3-compatible off-site target (keys live in credentials file).
type BackupDestination struct {
	ID       string `yaml:"id" json:"id"`
	Label    string `yaml:"label" json:"label"`
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Bucket   string `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
}

// DefaultBackupPolicy returns product defaults when policy fields are unset.
func DefaultBackupPolicy() BackupPolicy {
	inc := true
	return BackupPolicy{
		Schedule:          "0 3 * * *",
		ReplicationFactor: 1,
		DataRetentionDays: 14,
		ConfigKeepCount:   30,
		IncludeConfig:     &inc,
	}
}

// ResolvedBackupPolicy merges spec policy with defaults.
func ResolvedBackupPolicy(s Spec) BackupPolicy {
	d := DefaultBackupPolicy()
	p := s.Backup.Policy
	if strings.TrimSpace(p.Schedule) != "" {
		d.Schedule = strings.TrimSpace(p.Schedule)
	}
	if p.ReplicationFactor > 0 {
		d.ReplicationFactor = p.ReplicationFactor
	}
	if p.DataRetentionDays > 0 {
		d.DataRetentionDays = p.DataRetentionDays
	}
	if p.ConfigKeepCount > 0 {
		d.ConfigKeepCount = p.ConfigKeepCount
	}
	if p.IncludeConfig != nil {
		d.IncludeConfig = p.IncludeConfig
	}
	return d
}

// NormalizeBackup trims destinations, applies defaults, and syncs legacy backup.remote.
func NormalizeBackup(spec *Spec) {
	spec.Backup.Destinations = trimBackupDestinations(spec.Backup.Destinations)
	if len(spec.Backup.Destinations) == 0 && strings.TrimSpace(spec.Backup.Remote.Endpoint) != "" {
		spec.Backup.Destinations = []BackupDestination{{
			ID:       "primary",
			Label:    "Primary",
			Endpoint: strings.TrimSpace(spec.Backup.Remote.Endpoint),
			Enabled:  true,
		}}
	}
	bucket := spec.BackupBucketName()
	for i := range spec.Backup.Destinations {
		if strings.TrimSpace(spec.Backup.Destinations[i].Bucket) == "" {
			spec.Backup.Destinations[i].Bucket = bucket
		}
	}
	if len(spec.Backup.Destinations) > 0 {
		spec.Backup.Remote.Endpoint = spec.Backup.Destinations[0].Endpoint
	}
	policy := ResolvedBackupPolicy(*spec)
	spec.Backup.Policy = policy
}

func trimBackupDestinations(list []BackupDestination) []BackupDestination {
	out := make([]BackupDestination, 0, len(list))
	seen := make(map[string]struct{})
	for _, d := range list {
		id := strings.TrimSpace(d.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		d.ID = id
		d.Label = strings.TrimSpace(d.Label)
		if d.Label == "" {
			d.Label = id
		}
		d.Endpoint = strings.TrimSpace(d.Endpoint)
		d.Provider = strings.TrimSpace(d.Provider)
		d.Bucket = strings.TrimSpace(d.Bucket)
		out = append(out, d)
	}
	return out
}

// EffectiveDestinations returns configured targets (including legacy single-remote).
func EffectiveDestinations(s Spec) []BackupDestination {
	if len(s.Backup.Destinations) > 0 {
		return trimBackupDestinations(s.Backup.Destinations)
	}
	ep := strings.TrimSpace(s.Backup.Remote.Endpoint)
	if ep == "" {
		return nil
	}
	bucket := s.BackupBucketName()
	return []BackupDestination{{
		ID: "primary", Label: "Primary", Endpoint: ep, Bucket: bucket, Enabled: true,
	}}
}

// EnabledBackupDestinations returns destinations participating in upload and RF.
func EnabledBackupDestinations(s Spec) []BackupDestination {
	var out []BackupDestination
	for _, d := range EffectiveDestinations(s) {
		if d.Enabled && d.Endpoint != "" {
			out = append(out, d)
		}
	}
	return out
}

// DestinationBucket resolves bucket name for a destination.
func DestinationBucket(s Spec, d BackupDestination) string {
	if b := strings.TrimSpace(d.Bucket); b != "" {
		return b
	}
	return s.BackupBucketName()
}

// ValidateBackupPolicy checks replication factor against enabled destinations.
func ValidateBackupPolicy(s Spec) error {
	policy := ResolvedBackupPolicy(s)
	enabled := len(EnabledBackupDestinations(s))
	if enabled == 0 {
		return nil
	}
	rf := policy.ReplicationFactor
	if rf < 1 {
		rf = 1
	}
	if rf > enabled {
		return fmt.Errorf(
			"backup.policy.replication_factor (%d) exceeds enabled destinations (%d)",
			rf, enabled,
		)
	}
	return nil
}

// FindBackupDestination returns a destination by id.
func FindBackupDestination(s Spec, id string) (BackupDestination, bool) {
	id = strings.TrimSpace(id)
	for _, d := range EffectiveDestinations(s) {
		if d.ID == id {
			return d, true
		}
	}
	return BackupDestination{}, false
}

// PrimaryBackupDestinationID is the id of the first configured destination (canonical primary).
func PrimaryBackupDestinationID(s Spec) string {
	dests := EffectiveDestinations(s)
	if len(dests) == 0 {
		return ""
	}
	return dests[0].ID
}

// IsPrimaryBackupDestination reports whether id is the current primary (first destination).
func IsPrimaryBackupDestination(s Spec, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	return PrimaryBackupDestinationID(s) == id
}

// PromoteBackupDestination makes promoteID the primary (index 0).
// formerPrimaryID is the id assigned to the demoted previous primary (for credential/status migration).
// promotedFormerID is the promote target's id before promotion (empty if unchanged).
func PromoteBackupDestination(spec *Spec, promoteID string) (formerPrimaryID, promotedFormerID string, err error) {
	promoteID = strings.TrimSpace(promoteID)
	if promoteID == "" {
		return "", "", fmt.Errorf("destination id required")
	}
	dests := spec.Backup.Destinations
	if len(dests) == 0 {
		return "", "", fmt.Errorf("no destinations configured")
	}
	idx := -1
	for i, d := range dests {
		if d.ID == promoteID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", "", fmt.Errorf("destination %q not found", promoteID)
	}
	if idx == 0 {
		NormalizeBackup(spec)
		return "", "", nil
	}
	promoted := dests[idx]
	demoted := dests[0]
	promotedFormerID = promoted.ID
	formerPrimaryID = demoted.ID

	if promoted.ID != "primary" {
		if demoted.ID == "primary" {
			demoted.ID = promoted.ID
			formerPrimaryID = demoted.ID
		}
		promoted.ID = "primary"
	}

	newList := make([]BackupDestination, 0, len(dests))
	newList = append(newList, promoted)
	for i, d := range dests {
		if i == idx {
			continue
		}
		if i == 0 {
			newList = append(newList, demoted)
			continue
		}
		newList = append(newList, d)
	}
	spec.Backup.Destinations = newList
	NormalizeBackup(spec)
	return formerPrimaryID, promotedFormerID, nil
}