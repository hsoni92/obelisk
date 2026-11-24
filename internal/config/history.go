package config

import "path/filepath"

// HistoryPath is the append-only state transition log beside the cluster config.
func HistoryPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "state-history.jsonl")
}
