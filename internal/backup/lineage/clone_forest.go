package lineage

import (
	"fmt"
	"sort"
	"strings"
)

const (
	CloneScopeSnapshot = "snapshot"
	CloneScopeLineage  = "lineage"
)

// CopyJob is one object copy when duplicating a backup entry.
type CopyJob struct {
	SourceObjectKey string
	Node            Node
}

// CloneForestPlan describes a new catalog forest without mutating the source.
type CloneForestPlan struct {
	SourceKey    string
	TargetForest Forest
	Jobs         []CopyJob
}

// NewCloneInstanceID returns an unused instance id for a cloned backup entry.
func NewCloneInstanceID(idx Index, serviceType string) string {
	serviceType = strings.ToLower(strings.TrimSpace(serviceType))
	for {
		id := "clone-" + strings.Split(NewBackupID(), "-")[0]
		if _, ok := idx.Forests[InstanceKey(serviceType, id)]; !ok {
			return id
		}
	}
}

// PlanCloneForest builds metadata for a duplicate backup catalog entry (orphan clone forest).
func PlanCloneForest(
	idx Index,
	sourceType, sourceInstanceID, scope, snapshotBackupID, label string,
	objectKeyFn func(serviceType, cloneInstanceID, sourceObjectKey string, seq int) (string, error),
) (CloneForestPlan, error) {
	var plan CloneForestPlan
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	sourceInstanceID = strings.TrimSpace(sourceInstanceID)
	sourceKey := InstanceKey(sourceType, sourceInstanceID)
	plan.SourceKey = sourceKey

	srcForest, ok := idx.Forests[sourceKey]
	if !ok || len(srcForest.Nodes) == 0 {
		return plan, fmt.Errorf("no backups for %s", sourceKey)
	}

	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = CloneScopeSnapshot
	}
	if scope != CloneScopeSnapshot && scope != CloneScopeLineage {
		return plan, fmt.Errorf("scope must be %q or %q", CloneScopeSnapshot, CloneScopeLineage)
	}

	var toCopy []Node
	if scope == CloneScopeLineage {
		toCopy = liveNodesSorted(srcForest)
		if len(toCopy) == 0 {
			return plan, fmt.Errorf("no live backups to copy")
		}
	} else {
		snapshotBackupID = strings.TrimSpace(snapshotBackupID)
		if snapshotBackupID == "" {
			snapshotBackupID = srcForest.ActiveBackupID
		}
		if snapshotBackupID == "" {
			return plan, fmt.Errorf("no snapshot to copy (set backup_id or active backup)")
		}
		n, found := srcForest.Nodes[snapshotBackupID]
		if !found || n.Tombstone {
			return plan, fmt.Errorf("backup %q not found", snapshotBackupID)
		}
		toCopy = []Node{n}
	}

	cloneID := NewCloneInstanceID(idx, sourceType)
	label = strings.TrimSpace(label)
	if label == "" {
		label = "Clone of " + sourceKey
	}

	target := Forest{
		LineageID:      NewLineageID(),
		ServiceType:    sourceType,
		InstanceID:     cloneID,
		Orphan:         true,
		Clone:          true,
		CloneSourceKey: sourceKey,
		DisplayName:    label,
		Nodes:          map[string]Node{},
	}

	idMap := map[string]string{}
	for _, src := range toCopy {
		idMap[src.ID] = NewBackupID()
	}

	activeCloneID := ""
	if srcForest.ActiveBackupID != "" {
		if mapped, ok := idMap[srcForest.ActiveBackupID]; ok {
			activeCloneID = mapped
		}
	}
	if activeCloneID == "" && len(toCopy) > 0 {
		activeCloneID = idMap[toCopy[len(toCopy)-1].ID]
	}
	target.ActiveBackupID = activeCloneID

	for seq, src := range toCopy {
		newID := idMap[src.ID]
		parent := ""
		if src.ParentID != "" {
			if mapped, ok := idMap[src.ParentID]; ok {
				parent = mapped
			}
		}
		newObjKey, err := objectKeyFn(sourceType, cloneID, src.ObjectKey, seq)
		if err != nil {
			return plan, err
		}
		node := Node{
			ID:          newID,
			ParentID:    parent,
			CreatedAt:   src.CreatedAt,
			Trigger:     TriggerClone,
			ObjectKey:   newObjKey,
			ManifestKey: ManifestKeyForObject(newObjKey),
			Size:        src.Size,
			Witness:     src.Witness,
			Name:        src.Name,
		}
		target.Nodes[newID] = node
		plan.Jobs = append(plan.Jobs, CopyJob{
			SourceObjectKey: src.ObjectKey,
			Node:            node,
		})
	}

	plan.TargetForest = target
	return plan, nil
}

func liveNodesSorted(f Forest) []Node {
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
	return nodes
}

// PutForest replaces or inserts one forest in the index.
func PutForest(configPath string, f Forest) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	if idx.Forests == nil {
		idx.Forests = map[string]Forest{}
	}
	key := InstanceKey(f.ServiceType, f.InstanceID)
	if f.Nodes == nil {
		f.Nodes = map[string]Node{}
	}
	idx.Forests[key] = f
	return Save(configPath, idx)
}
