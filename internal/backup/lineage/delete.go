package lineage

import (
	"fmt"
	"time"
)

// CascadeDeleteResult lists tombstoned node ids for an instance forest.
type CascadeDeleteResult struct {
	DeletedIDs   []string `json:"deleted_ids"`
	ObjectKeys   []string `json:"object_keys"`
	ManifestKeys []string `json:"manifest_keys"`
}

// CollectDescendants returns backupID and all transitive children in creation order.
func CollectDescendants(f Forest, rootID string) ([]string, error) {
	if f.Nodes == nil {
		return nil, fmt.Errorf("backup %q not found", rootID)
	}
	if _, ok := f.Nodes[rootID]; !ok {
		return nil, fmt.Errorf("backup %q not found", rootID)
	}
	children := map[string][]string{}
	for id, n := range f.Nodes {
		if n.Tombstone {
			continue
		}
		if n.ParentID != "" {
			children[n.ParentID] = append(children[n.ParentID], id)
		}
	}
	var out []string
	var walk func(string)
	walk = func(id string) {
		out = append(out, id)
		for _, c := range children[id] {
			walk(c)
		}
	}
	walk(rootID)
	return out, nil
}

// CascadeDelete tombstones root and descendants. If root was active, active moves to parent when possible.
func CascadeDelete(configPath, serviceType, instanceID, rootID string) (CascadeDeleteResult, error) {
	var res CascadeDeleteResult
	idx, err := Load(configPath)
	if err != nil {
		return res, err
	}
	key := InstanceKey(serviceType, instanceID)
	f, ok := idx.Forests[key]
	if !ok {
		return res, fmt.Errorf("no backup lineage for %s", key)
	}
	ids, err := CollectDescendants(f, rootID)
	if err != nil {
		return res, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, id := range ids {
		n := f.Nodes[id]
		if n.Tombstone {
			continue
		}
		res.DeletedIDs = append(res.DeletedIDs, id)
		res.ObjectKeys = append(res.ObjectKeys, n.ObjectKey)
		res.ManifestKeys = append(res.ManifestKeys, n.ManifestKey)
		n.Tombstone = true
		n.TombstoneAt = now
		f.Nodes[id] = n
	}
	if f.ActiveBackupID == rootID {
		root := f.Nodes[rootID]
		if root.ParentID != "" {
			if p, ok := f.Nodes[root.ParentID]; ok && !p.Tombstone {
				f.ActiveBackupID = root.ParentID
			} else {
				f.ActiveBackupID = ""
			}
		} else {
			f.ActiveBackupID = ""
		}
	}
	idx.Forests[key] = f
	if err := Save(configPath, idx); err != nil {
		return res, err
	}
	return res, nil
}

// DeleteEntireForest removes all backup metadata for one instance forest and returns object keys to purge.
func DeleteEntireForest(configPath, serviceType, instanceID string) (CascadeDeleteResult, error) {
	var res CascadeDeleteResult
	idx, err := Load(configPath)
	if err != nil {
		return res, err
	}
	key := InstanceKey(serviceType, instanceID)
	f, ok := idx.Forests[key]
	if !ok {
		return res, fmt.Errorf("no backup lineage for %s", key)
	}
	for id, n := range f.Nodes {
		if n.Tombstone {
			continue
		}
		res.DeletedIDs = append(res.DeletedIDs, id)
		res.ObjectKeys = append(res.ObjectKeys, n.ObjectKey)
		res.ManifestKeys = append(res.ManifestKeys, n.ManifestKey)
	}
	delete(idx.Forests, key)
	if err := Save(configPath, idx); err != nil {
		return res, err
	}
	return res, nil
}