package lineage

// SetForestProvenance records deploy-from-backup source on a new lineage forest.
func SetForestProvenance(configPath, serviceType, instanceID, sourceBackupID string) error {
	idx, err := Load(configPath)
	if err != nil {
		return err
	}
	f := EnsureForest(&idx, serviceType, instanceID)
	f.SourceBackupID = sourceBackupID
	f.Orphan = false
	f.OrphanedAt = ""
	idx.Forests[InstanceKey(serviceType, instanceID)] = f
	return Save(configPath, idx)
}
