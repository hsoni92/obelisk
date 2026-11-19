package lineage

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// ManifestKeyForObject returns the sidecar key beside a data object.
func ManifestKeyForObject(objectKey string) string {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return ""
	}
	if strings.HasSuffix(objectKey, ManifestSuffix) {
		return objectKey
	}
	i := strings.LastIndex(objectKey, ".")
	if i < 0 {
		return objectKey + ManifestSuffix
	}
	return objectKey[:i] + ManifestSuffix
}

// NewManifest builds a sidecar for upload.
func NewManifest(node Node, forest Forest, machineID string) Manifest {
	return Manifest{
		SchemaVersion: 1,
		BackupID:      node.ID,
		LineageID:     forest.LineageID,
		ServiceType:   forest.ServiceType,
		InstanceID:    forest.InstanceID,
		ParentID:      node.ParentID,
		CreatedAt:     node.CreatedAt,
		Trigger:       node.Trigger,
		ObjectKey:     node.ObjectKey,
		Size:          node.Size,
		Witness:       node.Witness,
		MachineID:     machineID,
	}
}

func (m Manifest) JSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseManifest decodes a sidecar body.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// NewBackupID returns a random backup node id.
func NewBackupID() string {
	return uuid.NewString()
}

// NewLineageID returns a random forest id.
func NewLineageID() string {
	return uuid.NewString()
}