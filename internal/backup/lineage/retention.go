package lineage

import (
	"strings"
	"time"
)

// PruneReport summarizes retention dry-run or apply results.
type PruneReport struct {
	DryRun      bool     `json:"dry_run"`
	PrunedIDs   []string `json:"pruned_ids,omitempty"`
	ProtectedID string   `json:"protected_active_id,omitempty"`
}

// PruneRetention tombstones nodes older than retention days, never the active pointer.
func PruneRetention(configPath string, retentionDays int, dryRun bool) (PruneReport, error) {
	report := PruneReport{DryRun: dryRun}
	if retentionDays <= 0 {
		return report, nil
	}
	idx, err := Load(configPath)
	if err != nil {
		return report, err
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	changed := false
	for key, f := range idx.Forests {
		if f.Nodes == nil {
			continue
		}
		for id, n := range f.Nodes {
			if n.Tombstone {
				continue
			}
			if id == f.ActiveBackupID {
				report.ProtectedID = id
				continue
			}
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(n.CreatedAt))
			if err != nil {
				continue
			}
			if t.After(cutoff) {
				continue
			}
			report.PrunedIDs = append(report.PrunedIDs, id)
			if dryRun {
				continue
			}
			n.Tombstone = true
			n.TombstoneAt = time.Now().UTC().Format(time.RFC3339)
			f.Nodes[id] = n
			changed = true
		}
		idx.Forests[key] = f
	}
	if !dryRun && changed {
		if err := Save(configPath, idx); err != nil {
			return report, err
		}
	}
	return report, nil
}
