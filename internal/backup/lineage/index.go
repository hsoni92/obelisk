package lineage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// InstanceKey returns the canonical map key for a service instance.
func InstanceKey(serviceType, instanceID string) string {
	return strings.ToLower(strings.TrimSpace(serviceType)) + "/" + strings.TrimSpace(instanceID)
}

func indexPath(configPath string) string {
	return config.CredentialsDir(configPath) + "/" + IndexFileName
}

// Load reads backup-index.json or returns an empty index.
func Load(configPath string) (Index, error) {
	p := indexPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Index{Version: IndexVersion, Forests: map[string]Forest{}}, nil
		}
		return Index{}, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, err
	}
	if idx.Forests == nil {
		idx.Forests = map[string]Forest{}
	}
	if idx.Version == 0 {
		idx.Version = IndexVersion
	}
	return idx, nil
}

// Save writes backup-index.json atomically.
func Save(configPath string, idx Index) error {
	if idx.Forests == nil {
		idx.Forests = map[string]Forest{}
	}
	if idx.Version == 0 {
		idx.Version = IndexVersion
	}
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	p := indexPath(configPath)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// EnsureForest returns the forest for an instance, creating one when missing.
func EnsureForest(idx *Index, serviceType, instanceID string) Forest {
	key := InstanceKey(serviceType, instanceID)
	if idx.Forests == nil {
		idx.Forests = map[string]Forest{}
	}
	f, ok := idx.Forests[key]
	if !ok || f.LineageID == "" {
		f = Forest{
			LineageID:   NewLineageID(),
			ServiceType: strings.ToLower(strings.TrimSpace(serviceType)),
			InstanceID:  strings.TrimSpace(instanceID),
			Nodes:       map[string]Node{},
		}
		idx.Forests[key] = f
	}
	if f.Nodes == nil {
		f.Nodes = map[string]Node{}
		idx.Forests[key] = f
	}
	return f
}

// RegisterInput is produced after a successful data upload.
type RegisterInput struct {
	ServiceType string
	InstanceID  string
	ObjectKey   string
	Size        int64
	Trigger     string
	Witness     Witness
}

// PlanNode builds the next node without persisting (used before S3 upload).
func PlanNode(configPath string, in RegisterInput) (Node, Forest, error) {
	if in.Trigger == "" {
		in.Trigger = TriggerManual
	}
	idx, err := Load(configPath)
	if err != nil {
		return Node{}, Forest{}, err
	}
	f := EnsureForest(&idx, in.ServiceType, in.InstanceID)
	parentID := f.ActiveBackupID
	if parentID == "" {
		parentID = latestSuccessfulNodeID(f)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	node := Node{
		ID:          NewBackupID(),
		ParentID:    parentID,
		CreatedAt:   now,
		Trigger:     in.Trigger,
		ObjectKey:   in.ObjectKey,
		ManifestKey: ManifestKeyForObject(in.ObjectKey),
		Size:        in.Size,
		Witness:     in.Witness,
	}
	return node, f, nil
}

// CommitNode persists a planned node after a successful upload.
func CommitNode(configPath string, serviceType, instanceID string, node Node) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	key := InstanceKey(serviceType, instanceID)
	f := EnsureForest(&idx, serviceType, instanceID)
	f.Nodes[node.ID] = node
	f.ActiveBackupID = node.ID
	idx.Forests[key] = f
	return Save(configPath, idx)
}

// RegisterNode appends a child node, sets active pointer, and persists the index.
func RegisterNode(configPath string, in RegisterInput) (Node, error) {
	node, _, err := PlanNode(configPath, in)
	if err != nil {
		return Node{}, err
	}
	if err := CommitNode(configPath, in.ServiceType, in.InstanceID, node); err != nil {
		return Node{}, err
	}
	return node, nil
}

func latestSuccessfulNodeID(f Forest) string {
	type item struct {
		id string
		at string
	}
	var items []item
	for id, n := range f.Nodes {
		if n.Tombstone {
			continue
		}
		items = append(items, item{id: id, at: n.CreatedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].at < items[j].at
	})
	if len(items) == 0 {
		return ""
	}
	return items[len(items)-1].id
}

// ListNodes returns non-tombstoned nodes for an instance forest, oldest first.
func ListNodes(idx Index, serviceType, instanceID string) ([]Node, Forest, bool) {
	key := InstanceKey(serviceType, instanceID)
	f, ok := idx.Forests[key]
	if !ok {
		return nil, Forest{}, false
	}
	var nodes []Node
	for _, n := range f.Nodes {
		if n.Tombstone {
			continue
		}
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CreatedAt < nodes[j].CreatedAt
	})
	return nodes, f, true
}

// WitnessFromSpec builds capture-time witness from effective policy.
func WitnessFromSpec(spec obelisk.Spec, inst obelisk.ServiceInstance) Witness {
	view, _ := obelisk.ResolvedInstanceBackupPolicy(inst, spec.Backup.Policy)
	return Witness{
		Schedule:          view.Schedule,
		ReplicationFactor: view.ReplicationFactor,
		DataRetentionDays: view.DataRetentionDays,
	}
}

// MarkOrphan marks a forest as orphan when its instance row is removed.
func MarkOrphan(configPath, serviceType, instanceID string) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	key := InstanceKey(serviceType, instanceID)
	f, ok := idx.Forests[key]
	if !ok {
		return nil
	}
	f.Orphan = true
	f.OrphanedAt = time.Now().UTC().Format(time.RFC3339)
	idx.Forests[key] = f
	return Save(configPath, idx)
}

// MarkOrphanByKeys marks multiple instance forests orphan.
func MarkOrphanByKeys(configPath string, keys []string) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, key := range keys {
		f, ok := idx.Forests[key]
		if !ok {
			continue
		}
		f.Orphan = true
		f.OrphanedAt = now
		idx.Forests[key] = f
	}
	return Save(configPath, idx)
}

// RemoveForest deletes forest metadata (not S3 blobs).
func RemoveForest(configPath, serviceType, instanceID string) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	key := InstanceKey(serviceType, instanceID)
	if _, ok := idx.Forests[key]; !ok {
		return nil
	}
	delete(idx.Forests, key)
	return Save(configPath, idx)
}

// NodeByID looks up a node within an instance forest.
func NodeByID(idx Index, serviceType, instanceID, backupID string) (Node, Forest, bool) {
	key := InstanceKey(serviceType, instanceID)
	f, ok := idx.Forests[key]
	if !ok {
		return Node{}, Forest{}, false
	}
	n, ok := f.Nodes[backupID]
	if !ok || n.Tombstone {
		return Node{}, f, false
	}
	return n, f, true
}

// SetActiveBackup updates the active pointer for an instance forest.
func SetActiveBackup(configPath, serviceType, instanceID, backupID string) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	key := InstanceKey(serviceType, instanceID)
	f, ok := idx.Forests[key]
	if !ok {
		return fmt.Errorf("no backup lineage for %s", key)
	}
	n, ok := f.Nodes[backupID]
	if !ok || n.Tombstone {
		return fmt.Errorf("backup %q not found", backupID)
	}
	f.ActiveBackupID = backupID
	idx.Forests[key] = f
	return Save(configPath, idx)
}
