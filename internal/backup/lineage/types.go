package lineage

// IndexVersion is the backup-index.json schema version.
const IndexVersion = 1

// IndexFileName is stored beside obelisk.yaml under credentials/.
const IndexFileName = "backup-index.json"

// ManifestSuffix is the sidecar object name suffix (e.g. 20260101T120000.manifest.json).
const ManifestSuffix = ".manifest.json"

// TriggerManual and TriggerScheduled classify how a node was created.
const (
	TriggerManual    = "manual"
	TriggerScheduled = "scheduled"
	TriggerClone     = "clone"
)

// Witness captures backup policy fields at upload time (read-only on nodes).
type Witness struct {
	Schedule          string `json:"schedule"`
	ReplicationFactor int    `json:"replication_factor"`
	DataRetentionDays int    `json:"data_retention_days"`
}

// Node is one backup record in a service forest.
type Node struct {
	ID          string  `json:"id"`
	ParentID    string  `json:"parent_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
	Trigger     string  `json:"trigger"`
	ObjectKey   string  `json:"object_key"`
	ManifestKey string  `json:"manifest_key"`
	Size        int64   `json:"size"`
	Witness     Witness `json:"witness"`
	Name        string  `json:"name,omitempty"`
	Tombstone   bool    `json:"tombstone,omitempty"`
	TombstoneAt string  `json:"tombstone_at,omitempty"`
}

// Forest is the lineage tree for one service instance (or orphan lineage).
type Forest struct {
	LineageID      string          `json:"lineage_id"`
	ServiceType    string          `json:"service_type"`
	InstanceID     string          `json:"instance_id"`
	DisplayName    string          `json:"display_name,omitempty"`
	Orphan         bool            `json:"orphan,omitempty"`
	OrphanedAt     string          `json:"orphaned_at,omitempty"`
	ActiveBackupID string          `json:"active_backup_id,omitempty"`
	Nodes          map[string]Node `json:"nodes,omitempty"`
	SourceBackupID string          `json:"source_backup_id,omitempty"` // deploy-from-backup provenance
	Clone          bool            `json:"clone,omitempty"`
	CloneSourceKey string          `json:"clone_source_key,omitempty"`
}

// Index is the machine-local authority for backup metadata.
type Index struct {
	Version int               `json:"version"`
	Forests map[string]Forest `json:"forests,omitempty"` // key: instance key "type/id"
}

// Manifest is the S3 sidecar written next to each data object.
type Manifest struct {
	SchemaVersion int     `json:"schema_version"`
	BackupID      string  `json:"backup_id"`
	LineageID     string  `json:"lineage_id"`
	ServiceType   string  `json:"service_type"`
	InstanceID    string  `json:"instance_id"`
	ParentID      string  `json:"parent_id,omitempty"`
	CreatedAt     string  `json:"created_at"`
	Trigger       string  `json:"trigger"`
	ObjectKey     string  `json:"object_key"`
	Size          int64   `json:"size"`
	Witness       Witness `json:"witness"`
	MachineID     string  `json:"machine_id,omitempty"`
}
