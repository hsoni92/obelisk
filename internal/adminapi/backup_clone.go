package adminapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/backup/lineage"
	"github.com/hsoni92/obelisk/internal/config"
)

type cloneBackupRequest struct {
	SourceType       string `json:"source_type"`
	SourceInstanceID string `json:"source_instance_id"`
	Scope            string `json:"scope"` // snapshot | lineage
	BackupID         string `json:"backup_id,omitempty"`
	Label            string `json:"label,omitempty"`
}

func (s *Server) handlePostCloneBackup(w http.ResponseWriter, r *http.Request) {
	var req cloneBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.SourceType = strings.TrimSpace(req.SourceType)
	req.SourceInstanceID = strings.TrimSpace(req.SourceInstanceID)
	req.Scope = strings.TrimSpace(req.Scope)
	req.BackupID = strings.TrimSpace(req.BackupID)
	if req.SourceType == "" || req.SourceInstanceID == "" {
		http.Error(w, "source_type and source_instance_id are required", http.StatusBadRequest)
		return
	}
	if req.Scope == "" {
		req.Scope = lineage.CloneScopeSnapshot
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := backup.CloneBackupEntry(
		r.Context(),
		cfg,
		spec,
		req.SourceType,
		req.SourceInstanceID,
		req.Scope,
		req.BackupID,
		req.Label,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.recordAudit(
		"Backup",
		"cloned",
		out.SourceKey+" → "+out.ServiceType+"/"+out.InstanceID+" ("+out.Scope+")",
	)

	writeJSON(w, map[string]any{
		"status":                "ok",
		"message":               "backup entry cloned",
		"scope":                 out.Scope,
		"source_key":            out.SourceKey,
		"service_type":          out.ServiceType,
		"instance_id":           out.InstanceID,
		"lineage_id":            out.LineageID,
		"node_count":            out.NodeCount,
		"uploaded_destinations": out.UploadedDestinations,
	})
}
