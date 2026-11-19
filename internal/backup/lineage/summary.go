package lineage

// ForestSummary is a compact view for catalog / monitor UIs.
type ForestSummary struct {
	NodeCount       int
	ActiveID        string
	ActiveCreatedAt string
	ActiveSize      int64
}

// SummarizeForest counts live nodes and resolves the active snapshot metadata.
func SummarizeForest(f Forest) ForestSummary {
	var out ForestSummary
	for _, n := range f.Nodes {
		if n.Tombstone {
			continue
		}
		out.NodeCount++
	}
	out.ActiveID = f.ActiveBackupID
	if out.ActiveID != "" {
		if n, ok := f.Nodes[out.ActiveID]; ok && !n.Tombstone {
			out.ActiveCreatedAt = n.CreatedAt
			out.ActiveSize = n.Size
		}
	}
	return out
}