package lineage

import (
	"path"
	"sort"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// dataObjectBasename extracts the filename portion of a data/ object key.
func dataObjectBasename(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return path.Base(key)
}

// isDataBackupObject reports whether key is a legacy data dump (not manifest).
func isDataBackupObject(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" || !strings.HasPrefix(key, obelisk.BackupDataPrefix) {
		return false
	}
	if strings.HasSuffix(key, ManifestSuffix) {
		return false
	}
	base := dataObjectBasename(key)
	if base == "" || base == "." {
		return false
	}
	return true
}

// parseInstanceFromDataKey returns service type and id from data/<type>/<id>/...
func parseInstanceFromDataKey(key string) (typ, id string, ok bool) {
	key = strings.TrimPrefix(strings.TrimSpace(key), obelisk.BackupDataPrefix)
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// BackfillFromKeys synthesizes nodes for legacy uploads without manifests.
func BackfillFromKeys(configPath string, keys []string, witness Witness) (int, error) {
	idx, err := Load(configPath)
	if err != nil {
		return 0, err
	}
	type stamp struct {
		key string
		ts  time.Time
	}
	byInst := map[string][]stamp{}
	for _, k := range keys {
		if !isDataBackupObject(k) {
			continue
		}
		typ, id, ok := parseInstanceFromDataKey(k)
		if !ok {
			continue
		}
		instKey := InstanceKey(typ, id)
		base := strings.TrimSuffix(dataObjectBasename(k), path.Ext(k))
		t, _ := time.Parse("20060102T150405Z", base)
		byInst[instKey] = append(byInst[instKey], stamp{key: k, ts: t})
	}
	added := 0
	for instKey, stamps := range byInst {
		sort.Slice(stamps, func(i, j int) bool {
			return stamps[i].ts.Before(stamps[j].ts)
		})
		parts := strings.SplitN(instKey, "/", 2)
		if len(parts) != 2 {
			continue
		}
		f := EnsureForest(&idx, parts[0], parts[1])
		if len(f.Nodes) > 0 {
			idx.Forests[instKey] = f
			continue
		}
		var parent string
		for _, s := range stamps {
			created := s.ts.UTC().Format(time.RFC3339)
			if created == "0001-01-01T00:00:00Z" {
				created = time.Now().UTC().Format(time.RFC3339)
			}
			id := NewBackupID()
			node := Node{
				ID:          id,
				ParentID:    parent,
				CreatedAt:   created,
				Trigger:     TriggerManual,
				ObjectKey:   s.key,
				ManifestKey: ManifestKeyForObject(s.key),
				Witness:     witness,
			}
			f.Nodes[id] = node
			f.ActiveBackupID = id
			parent = id
			added++
		}
		idx.Forests[instKey] = f
	}
	if added == 0 {
		return 0, nil
	}
	return added, Save(configPath, idx)
}
