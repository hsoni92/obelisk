package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/backup/lineage"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type backupNodeView struct {
	ID          string         `json:"id"`
	ParentID    string         `json:"parent_id,omitempty"`
	CreatedAt   string         `json:"created_at"`
	Trigger     string         `json:"trigger"`
	ObjectKey   string         `json:"object_key"`
	ManifestKey string         `json:"manifest_key"`
	Size        int64          `json:"size"`
	Witness     lineage.Witness `json:"witness"`
	Active      bool           `json:"active"`
}

type instanceBackupLineageResponse struct {
	ServiceType      string           `json:"service_type"`
	InstanceID       string           `json:"instance_id"`
	LineageID        string           `json:"lineage_id"`
	DisplayName      string           `json:"display_name,omitempty"`
	Orphan           bool             `json:"orphan"`
	Clone            bool             `json:"clone,omitempty"`
	CloneSourceKey   string           `json:"clone_source_key,omitempty"`
	ActiveBackupID   string           `json:"active_backup_id,omitempty"`
	SourceBackupID   string           `json:"source_backup_id,omitempty"`
	Nodes            []backupNodeView `json:"nodes"`
}

func parseInstanceQuery(r *http.Request) (typ, id string, err error) {
	id = strings.TrimSpace(r.PathValue("id"))
	typ = strings.TrimSpace(r.URL.Query().Get("type"))
	if id == "" {
		return "", "", fmt.Errorf("instance id required")
	}
	if typ == "" {
		return "", "", fmt.Errorf("query type= is required")
	}
	return typ, id, nil
}

func (s *Server) handleGetInstanceBackups(w http.ResponseWriter, r *http.Request) {
	typ, id, err := parseInstanceQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	idx, err := lineage.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nodes, forest, ok := lineage.ListNodes(idx, typ, id)
	if !ok {
		writeJSON(w, instanceBackupLineageResponse{
			ServiceType: typ,
			InstanceID:  id,
			Nodes:       []backupNodeView{},
		})
		return
	}
	var views []backupNodeView
	for _, n := range nodes {
		views = append(views, backupNodeView{
			ID:          n.ID,
			ParentID:    n.ParentID,
			CreatedAt:   n.CreatedAt,
			Trigger:     n.Trigger,
			ObjectKey:   n.ObjectKey,
			ManifestKey: n.ManifestKey,
			Size:        n.Size,
			Witness:     n.Witness,
			Active:      n.ID == forest.ActiveBackupID,
		})
	}
	writeJSON(w, instanceBackupLineageResponse{
		ServiceType:      typ,
		InstanceID:       id,
		LineageID:        forest.LineageID,
		DisplayName:      forest.DisplayName,
		Orphan:           forest.Orphan,
		Clone:            forest.Clone,
		CloneSourceKey:   forest.CloneSourceKey,
		ActiveBackupID:   forest.ActiveBackupID,
		SourceBackupID:   forest.SourceBackupID,
		Nodes:            views,
	})
}

func (s *Server) handlePostInstanceBackupRun(w http.ResponseWriter, r *http.Request) {
	typ, id, err := parseInstanceQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !obelisk.ServiceTypeSupportsBackup(typ) {
		http.Error(w, "service type does not support data backup", http.StatusBadRequest)
		return
	}
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	states := rec.ObserveServiceStates(r.Context(), spec)
	runner := backup.Runner{ConfigPath: s.ConfigPath, InstanceStates: states}
	out, err := runner.RunForInstance(r.Context(), typ, id)
	if err != nil {
		if out.UploadedDestinations > 0 {
			writeJSON(w, map[string]any{
				"status":  "degraded",
				"message": err.Error(),
				"backups": out.RegisteredBackups,
			})
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.recordAudit("Backup", "instance", typ+"/"+id)
	writeJSON(w, map[string]any{
		"status":      "ok",
		"backups":     out.RegisteredBackups,
		"data_objects": out.DataObjects,
	})
}

type deleteBackupRequest struct {
	Confirm string `json:"confirm"`
}

func (s *Server) handleDeleteInstanceBackupForest(w http.ResponseWriter, r *http.Request) {
	typ, id, err := parseInstanceQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req deleteBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	want := strings.ToUpper(strings.TrimSpace(typ + "/" + id))
	if strings.ToUpper(strings.TrimSpace(req.Confirm)) != want {
		http.Error(w, fmt.Sprintf("type %q in confirm body", want), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := lineage.DeleteEntireForest(cfg, typ, id)
	if err != nil {
		if strings.Contains(err.Error(), "no backup lineage") {
			_ = lineage.RemoveForest(cfg, typ, id)
			res = lineage.CascadeDeleteResult{}
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	allKeys := uniqueNonempty(append(append([]string{}, res.ObjectKeys...), res.ManifestKeys...))
	prefixKeys, listErr := backup.ListInstanceBackupObjectKeys(ctx, cfg, spec, typ, id)
	if listErr != nil {
		writeJSON(w, map[string]any{
			"status":  "degraded",
			"deleted": res.DeletedIDs,
			"message": listErr.Error(),
		})
		return
	}
	allKeys = uniqueNonempty(append(allKeys, prefixKeys...))
	if len(allKeys) == 0 {
		s.recordAudit("Backup", "forest_deleted", typ+"/"+id)
		writeJSON(w, map[string]any{"status": "ok", "deleted": res.DeletedIDs, "objects_purged": 0})
		return
	}
	if err := backup.DeleteObjectsAllDestinations(ctx, cfg, spec, allKeys); err != nil {
		writeJSON(w, map[string]any{
			"status":  "degraded",
			"deleted": res.DeletedIDs,
			"message": err.Error(),
		})
		return
	}
	s.recordAudit("Backup", "forest_deleted", typ+"/"+id)
	writeJSON(w, map[string]any{
		"status":         "ok",
		"deleted":        res.DeletedIDs,
		"objects_purged": len(allKeys),
	})
}

func uniqueNonempty(keys []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func (s *Server) handleDeleteInstanceBackup(w http.ResponseWriter, r *http.Request) {
	typ, id, err := parseInstanceQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	backupID := strings.TrimSpace(r.PathValue("backupId"))
	if backupID == "" {
		http.Error(w, "backup id required", http.StatusBadRequest)
		return
	}
	var req deleteBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	want := strings.ToUpper(strings.TrimSpace(typ + "/" + id))
	if strings.ToUpper(strings.TrimSpace(req.Confirm)) != want {
		http.Error(w, fmt.Sprintf("type %q in confirm body", want), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := lineage.CascadeDelete(cfg, typ, id, backupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	allKeys := append(append([]string{}, res.ObjectKeys...), res.ManifestKeys...)
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := backup.DeleteObjectsAllDestinations(ctx, cfg, spec, allKeys); err != nil {
		writeJSON(w, map[string]any{
			"status":  "degraded",
			"deleted": res.DeletedIDs,
			"message": err.Error(),
		})
		return
	}
	s.recordAudit("Backup", "deleted", typ+"/"+id+" "+backupID)
	writeJSON(w, map[string]any{"status": "ok", "deleted": res.DeletedIDs})
}

func (s *Server) handlePostInstanceRestoreActive(w http.ResponseWriter, r *http.Request) {
	typ, id, err := parseInstanceQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !obelisk.ServiceTypeSupportsBackup(typ) {
		http.Error(w, "service type does not support data backup", http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	inst, _, ok := obelisk.FindInstanceByTypeAndID(&spec, typ, id)
	if !ok {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	idx, err := lineage.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, forest, ok := lineage.ListNodes(idx, typ, id)
	if !ok || forest.ActiveBackupID == "" {
		http.Error(w, "no active backup", http.StatusBadRequest)
		return
	}
	node, _, ok := lineage.NodeByID(idx, typ, id, forest.ActiveBackupID)
	if !ok {
		http.Error(w, "active backup missing", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	var dump []byte
	var fetchErr error
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := backup.LoadDestinationCredentials(cfg, d.ID)
		if err != nil || !ok {
			legacy, legOK, _ := backup.LoadCredentials(cfg)
			if legOK {
				creds = legacy
				ok = true
			}
		}
		if !ok {
			continue
		}
		bucket := obelisk.DestinationBucket(spec, d)
		dump, fetchErr = backup.GetObject(ctx, d.Endpoint, bucket, node.ObjectKey, creds)
		if fetchErr == nil {
			break
		}
	}
	if fetchErr != nil {
		http.Error(w, fetchErr.Error(), http.StatusBadGateway)
		return
	}
	if err := backup.RestoreInstanceData(ctx, inst, dump); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.recordAudit("Backup", "restored", typ+"/"+id+" active="+node.ID)
	writeJSON(w, map[string]any{"status": "ok", "backup_id": node.ID})
}

type backupCatalogRow struct {
	ServiceType            string `json:"service_type"`
	InstanceID             string `json:"instance_id"`
	DisplayName            string `json:"display_name,omitempty"`
	InstanceLive           bool   `json:"instance_live"`
	BackupEnabled          bool   `json:"backup_enabled"`
	Orphan                 bool   `json:"orphan"`
	State                  string `json:"state,omitempty"`
	BackupCount            int    `json:"backup_count"`
	ActiveBackupID         string `json:"active_backup_id,omitempty"`
	ActiveBackupAt         string `json:"active_backup_at,omitempty"`
	ActiveBackupBytes      int64  `json:"active_backup_bytes,omitempty"`
	LastUploadAt           string `json:"last_upload_at,omitempty"`
	LineageID              string `json:"lineage_id,omitempty"`
	Clone                  bool   `json:"clone,omitempty"`
	CloneSourceKey         string `json:"clone_source_key,omitempty"`
	ReplicationFactor      int    `json:"replication_factor,omitempty"`
	ReplicationOverridden  bool   `json:"replication_overridden,omitempty"`
	ReplicationFromWitness bool   `json:"replication_from_witness,omitempty"`
}

type backupCatalogResponse struct {
	GeneratedAt               string             `json:"generated_at"`
	GlobalReplicationFactor   int                `json:"global_replication_factor"`
	EnabledDestinationCount   int                `json:"enabled_destination_count"`
	ReadyDestinationCount     int                `json:"ready_destination_count"`
	ReplicationReady          bool               `json:"replication_ready"`
	Rows                      []backupCatalogRow `json:"rows"`
}

func (s *Server) handleGetBackupCatalog(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rec := s.reconciler(cfg)
	states := rec.ObserveServiceStates(r.Context(), spec)
	status, _ := backup.LoadStatus(cfg)
	idx, err := lineage.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	globalPolicy := obelisk.ResolvedBackupPolicy(spec)
	globalRF := globalPolicy.ReplicationFactor
	if globalRF < 1 {
		globalRF = 1
	}
	enabledDests := len(obelisk.EnabledBackupDestinations(spec))
	readyDests := countReadyBackupDestinations(cfg, spec, status)

	liveKeys := map[string]struct{}{}
	var rows []backupCatalogRow

	addRow := func(row backupCatalogRow) {
		rows = append(rows, row)
	}

	for _, inst := range spec.Instances {
		if !obelisk.ServiceTypeSupportsBackup(inst.Type) {
			continue
		}
		key := lineage.InstanceKey(inst.Type, inst.ID)
		liveKeys[key] = struct{}{}
		instKey := inst.Type + "/" + inst.ID
		view, _ := obelisk.ResolvedInstanceBackupPolicy(inst, spec.Backup.Policy)
		effRF := view.ReplicationFactor
		if effRF < 1 {
			effRF = 1
		}
		row := backupCatalogRow{
			ServiceType:           inst.Type,
			InstanceID:            inst.ID,
			DisplayName:           inst.DisplayName,
			InstanceLive:          true,
			BackupEnabled:         obelisk.InstanceDataBackupEnabled(inst),
			State:                 states[instKey],
			LastUploadAt:          status.Instances[instKey],
			ReplicationFactor:     effRF,
			ReplicationOverridden: effRF != globalRF,
		}
		if f, ok := idx.Forests[key]; ok {
			sum := lineage.SummarizeForest(f)
			row.Orphan = f.Orphan
			row.LineageID = f.LineageID
			row.Clone = f.Clone
			row.CloneSourceKey = f.CloneSourceKey
			if f.DisplayName != "" {
				row.DisplayName = f.DisplayName
			}
			row.BackupCount = sum.NodeCount
			row.ActiveBackupID = sum.ActiveID
			row.ActiveBackupAt = sum.ActiveCreatedAt
			row.ActiveBackupBytes = sum.ActiveSize
		}
		addRow(row)
	}

	for key, f := range idx.Forests {
		if _, live := liveKeys[key]; live {
			continue
		}
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			continue
		}
		sum := lineage.SummarizeForest(f)
		state := "orphan"
		if f.Clone {
			state = "clone"
		}
		orphanRow := backupCatalogRow{
			ServiceType:       parts[0],
			InstanceID:        parts[1],
			DisplayName:       f.DisplayName,
			InstanceLive:      false,
			Orphan:            true,
			Clone:             f.Clone,
			CloneSourceKey:    f.CloneSourceKey,
			BackupEnabled:     false,
			BackupCount:       sum.NodeCount,
			ActiveBackupID:    sum.ActiveID,
			ActiveBackupAt:    sum.ActiveCreatedAt,
			ActiveBackupBytes: sum.ActiveSize,
			LastUploadAt:      status.Instances[key],
			LineageID:         f.LineageID,
			State:             state,
		}
		if sum.ActiveWitnessRF > 0 {
			orphanRow.ReplicationFactor = sum.ActiveWitnessRF
			orphanRow.ReplicationFromWitness = true
		}
		addRow(orphanRow)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ServiceType != rows[j].ServiceType {
			return rows[i].ServiceType < rows[j].ServiceType
		}
		return rows[i].InstanceID < rows[j].InstanceID
	})

	writeJSON(w, backupCatalogResponse{
		GeneratedAt:             time.Now().UTC().Format(time.RFC3339),
		GlobalReplicationFactor: globalRF,
		EnabledDestinationCount: enabledDests,
		ReadyDestinationCount:   readyDests,
		ReplicationReady:        readyDests >= globalRF,
		Rows:                    rows,
	})
}

func countReadyBackupDestinations(cfg string, spec obelisk.Spec, status backup.StatusFile) int {
	ready := 0
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		credOK, _ := backup.DestinationCredentialsConfigured(cfg, d.ID)
		if !credOK {
			legacy, ok, _ := backup.LoadCredentials(cfg)
			if ok && d.ID == "primary" {
				credOK = legacy.AccessKeyID != ""
			}
		}
		st := status.Destinations[d.ID]
		if d.Enabled && d.Endpoint != "" && credOK && st.LastTestOK != nil && *st.LastTestOK {
			ready++
		}
	}
	return ready
}

func (s *Server) handlePostBackupIndexRebuild(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	n, err := backup.RebuildLineageIndex(ctx, cfg, spec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "nodes_added": n})
}