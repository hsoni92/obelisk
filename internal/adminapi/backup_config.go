package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type backupPolicyView struct {
	Schedule          string `json:"schedule"`
	ReplicationFactor int    `json:"replication_factor"`
	DataRetentionDays int    `json:"data_retention_days"`
	ConfigKeepCount   int    `json:"config_keep_count"`
	IncludeConfig     bool   `json:"include_config"`
}

type backupDestinationView struct {
	ID                    string `json:"id"`
	IsPrimary             bool   `json:"is_primary"`
	Label                 string `json:"label"`
	Provider              string `json:"provider,omitempty"`
	Endpoint              string `json:"endpoint"`
	Bucket                string `json:"bucket"`
	Enabled               bool   `json:"enabled"`
	CredentialsConfigured bool   `json:"credentials_configured"`
	Ready                 bool   `json:"ready"`
	Status                string `json:"status"`
	LastTestAt            string `json:"last_test_at,omitempty"`
	LastTestOK            *bool  `json:"last_test_ok,omitempty"`
	LastTestMessage       string `json:"last_test_message,omitempty"`
	LastUploadAt          string `json:"last_upload_at,omitempty"`
	LastInventoryAt       string `json:"last_inventory_at,omitempty"`
	Bytes                 int64  `json:"bytes,omitempty"`
	Objects               int    `json:"objects,omitempty"`
	LastError             string `json:"last_error,omitempty"`
	ScheduleLag           string `json:"schedule_lag,omitempty"`
}

type backupScopeView struct {
	DataServiceTypes []string `json:"data_service_types"`
}

type backupConfigResponse struct {
	MachineID              string                  `json:"machine_id"`
	Layout                 backupLayoutView        `json:"layout"`
	Scope                  backupScopeView         `json:"scope"`
	Policy                 backupPolicyView        `json:"policy"`
	Destinations           []backupDestinationView `json:"destinations"`
	BackupReady            bool                    `json:"backup_ready"`
	ReplicationReady       bool                    `json:"replication_ready"`
	ReadyDestinationCount  int                     `json:"ready_destination_count"`
	EnabledDestinationCount int                    `json:"enabled_destination_count"`
	ImplementationStatus   string                  `json:"implementation_status"`
	LastRunAt              string                  `json:"last_run_at,omitempty"`
	// LastBackupByInstance maps "<type>/<id>" to the RFC3339 time of its last data upload.
	LastBackupByInstance   map[string]string       `json:"last_backup_by_instance,omitempty"`
	CredentialsSecret      string                  `json:"credentials_secret"`
}

type patchBackupPolicyRequest struct {
	Schedule          *string `json:"schedule,omitempty"`
	ReplicationFactor *int    `json:"replication_factor,omitempty"`
	DataRetentionDays *int    `json:"data_retention_days,omitempty"`
	ConfigKeepCount   *int    `json:"config_keep_count,omitempty"`
	IncludeConfig     *bool   `json:"include_config,omitempty"`
}

type postBackupDestinationRequest struct {
	Label       string `json:"label"`
	Provider    string `json:"provider,omitempty"`
	Endpoint    string `json:"endpoint"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Credentials *backup.RemoteCredentials `json:"credentials,omitempty"`
}

type deleteBackupDestinationRequest struct {
	AcknowledgePrimaryRemoval bool   `json:"acknowledge_primary_removal"`
	PromoteDestinationID      string `json:"promote_destination_id,omitempty"`
}

type patchBackupDestinationRequest struct {
	Label       *string `json:"label,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	Endpoint    *string `json:"endpoint,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	Credentials *backup.RemoteCredentials `json:"credentials,omitempty"`
}

func (s *Server) handleGetBackupConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}

func (s *Server) buildBackupConfig(cfg string, spec obelisk.Spec) backupConfigResponse {
	_ = backup.MigrateLegacyCredentials(cfg, "primary")
	policy := obelisk.ResolvedBackupPolicy(spec)
	status, _ := backup.LoadStatus(cfg)
	secret := spec.Backup.CredentialsSecret
	if secret == "" {
		secret = obelisk.DefaultBackupCredentialsSecret
	}
	bucket := spec.BackupBucketName()
	dests := obelisk.EffectiveDestinations(spec)
	primaryID := obelisk.PrimaryBackupDestinationID(spec)
	var views []backupDestinationView
	readyCount := 0
	enabledCount := 0
	for _, d := range dests {
		if d.Enabled {
			enabledCount++
		}
		credOK, _ := backup.DestinationCredentialsConfigured(cfg, d.ID)
		if !credOK {
			legacy, ok, _ := backup.LoadCredentials(cfg)
			if ok && d.ID == "primary" {
				credOK = legacy.AccessKeyID != ""
			}
		}
		st := status.Destinations[d.ID]
		ready := d.Enabled && d.Endpoint != "" && credOK && st.LastTestOK != nil && *st.LastTestOK
		if ready {
			readyCount++
		}
		lag := scheduleLag(policy.Schedule, st.LastUploadAt, st.LastTestAt)
		views = append(views, backupDestinationView{
			ID:                    d.ID,
			IsPrimary:             d.ID == primaryID,
			Label:                 d.Label,
			Provider:              d.Provider,
			Endpoint:              d.Endpoint,
			Bucket:                obelisk.DestinationBucket(spec, d),
			Enabled:               d.Enabled,
			CredentialsConfigured: credOK,
			Ready:                 ready,
			Status:                destinationStatus(d, credOK, st),
			LastTestAt:            st.LastTestAt,
			LastTestOK:            st.LastTestOK,
			LastTestMessage:       st.LastTestMessage,
			LastUploadAt:          st.LastUploadAt,
			LastInventoryAt:       st.LastInventoryAt,
			Bytes:                 st.Bytes,
			Objects:               st.Objects,
			LastError:             st.LastError,
			ScheduleLag:           lag,
		})
	}
	rf := policy.ReplicationFactor
	if rf < 1 {
		rf = 1
	}
	replicationReady := readyCount >= rf
	backupReady := spec.BackupRemoteReady() && readyCount >= 1
	inc := true
	if policy.IncludeConfig != nil {
		inc = *policy.IncludeConfig
	}
	if views == nil {
		views = []backupDestinationView{}
	}
	return backupConfigResponse{
		MachineID: bucket,
		Scope: backupScopeView{
			DataServiceTypes: append([]string(nil), obelisk.BackupDataServiceTypes...),
		},
		Layout: backupLayoutView{
			Bucket:       bucket,
			ConfigPrefix: obelisk.BackupConfigPrefix,
			DataPrefix:   obelisk.BackupDataPrefix,
			PreflightKey: obelisk.BackupPreflightObject,
		},
		Policy: backupPolicyView{
			Schedule:          policy.Schedule,
			ReplicationFactor: policy.ReplicationFactor,
			DataRetentionDays: policy.DataRetentionDays,
			ConfigKeepCount:   policy.ConfigKeepCount,
			IncludeConfig:     inc,
		},
		Destinations:            views,
		BackupReady:             backupReady,
		ReplicationReady:        replicationReady,
		ReadyDestinationCount:   readyCount,
		EnabledDestinationCount: enabledCount,
		ImplementationStatus:    "stub",
		LastRunAt:               status.LastRunAt,
		LastBackupByInstance:    status.Instances,
		CredentialsSecret:       secret,
	}
}

func destinationStatus(d obelisk.BackupDestination, credOK bool, st backup.DestinationRuntimeStatus) string {
	if !d.Enabled {
		return "disabled"
	}
	if d.Endpoint == "" {
		return "unconfigured"
	}
	if !credOK {
		return "needs_keys"
	}
	if st.LastTestOK == nil {
		return "untested"
	}
	if !*st.LastTestOK {
		return "offline"
	}
	return "ready"
}

func scheduleLag(schedule, lastUpload, lastTest string) string {
	ref := lastUpload
	if ref == "" {
		ref = lastTest
	}
	if ref == "" {
		return "no backup yet"
	}
	t, err := time.Parse(time.RFC3339, ref)
	if err != nil {
		return ""
	}
	age := time.Since(t)
	switch {
	case strings.Contains(schedule, "*/6"):
		if age > 8*time.Hour {
			return fmt.Sprintf("%.0fh overdue", age.Hours())
		}
	case strings.Contains(schedule, "0 3"):
		if age > 36*time.Hour {
			return fmt.Sprintf("%.0fh since last", age.Hours())
		}
	default:
		if age > 48*time.Hour {
			return fmt.Sprintf("%.0fh since last", age.Hours())
		}
	}
	if age < 2*time.Hour {
		return "on schedule"
	}
	return fmt.Sprintf("%.0fh ago", age.Hours())
}

func (s *Server) handlePatchBackupPolicy(w http.ResponseWriter, r *http.Request) {
	var req patchBackupPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := spec.Backup.Policy
	if req.Schedule != nil {
		p.Schedule = strings.TrimSpace(*req.Schedule)
	}
	if req.ReplicationFactor != nil {
		p.ReplicationFactor = *req.ReplicationFactor
	}
	if req.DataRetentionDays != nil {
		p.DataRetentionDays = *req.DataRetentionDays
	}
	if req.ConfigKeepCount != nil {
		p.ConfigKeepCount = *req.ConfigKeepCount
	}
	if req.IncludeConfig != nil {
		p.IncludeConfig = req.IncludeConfig
	}
	spec.Backup.Policy = p
	if err := obelisk.ValidateBackupPolicy(spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}

func (s *Server) handlePostBackupDestination(w http.ResponseWriter, r *http.Request) {
	var req postBackupDestinationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	endpoint := strings.TrimSpace(req.Endpoint)
	if endpoint == "" {
		http.Error(w, "endpoint required", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "destination"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id := fmt.Sprintf("bd-%d", time.Now().UnixNano())
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	obelisk.EnsureMachineIdentity(&spec)
	dest := obelisk.BackupDestination{
		ID: id, Label: label, Provider: strings.TrimSpace(req.Provider),
		Endpoint: endpoint, Enabled: enabled, Bucket: spec.BackupBucketName(),
	}
	spec.Backup.Destinations = append(spec.Backup.Destinations, dest)
	if req.Credentials != nil && req.Credentials.AccessKeyID != "" {
		if err := backup.SaveDestinationCredentials(cfg, id, *req.Credentials); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if err := obelisk.ValidateBackupPolicy(spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfgResp := s.buildBackupConfig(cfg, spec)
	writeJSON(w, cfgResp)
}

func (s *Server) handlePatchBackupDestination(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	var req patchBackupDestinationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	idx := -1
	for i, d := range spec.Backup.Destinations {
		if d.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		http.Error(w, "destination not found", http.StatusNotFound)
		return
	}
	d := spec.Backup.Destinations[idx]
	if req.Label != nil {
		d.Label = strings.TrimSpace(*req.Label)
	}
	if req.Provider != nil {
		d.Provider = strings.TrimSpace(*req.Provider)
	}
	if req.Endpoint != nil {
		d.Endpoint = strings.TrimSpace(*req.Endpoint)
	}
	if req.Enabled != nil {
		d.Enabled = *req.Enabled
	}
	spec.Backup.Destinations[idx] = d
	if req.Credentials != nil && req.Credentials.AccessKeyID != "" {
		if err := backup.SaveDestinationCredentials(cfg, id, *req.Credentials); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if err := obelisk.ValidateBackupPolicy(spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}

func (s *Server) handleDeleteBackupDestination(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	var req deleteBackupDestinationRequest
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	isPrimary := obelisk.IsPrimaryBackupDestination(spec, id)
	destCount := len(obelisk.EffectiveDestinations(spec))
	if isPrimary {
		if !req.AcknowledgePrimaryRemoval {
			http.Error(w, "removing the primary destination requires acknowledge_primary_removal", http.StatusBadRequest)
			return
		}
		if destCount > 1 {
			promoteID := strings.TrimSpace(req.PromoteDestinationID)
			if promoteID == "" {
				http.Error(w, "promote_destination_id required when removing primary with other destinations configured", http.StatusBadRequest)
				return
			}
			if promoteID == id {
				http.Error(w, "choose a different destination to promote", http.StatusBadRequest)
				return
			}
			formerPrimaryID, promotedFormerID, err := obelisk.PromoteBackupDestination(&spec, promoteID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if promotedFormerID != "" && promotedFormerID != "primary" {
				_ = backup.SwapDestinationCredentials(cfg, "primary", promotedFormerID)
				_ = backup.SwapDestinationStatus(cfg, "primary", promotedFormerID)
			}
			id = formerPrimaryID
		}
	}
	var next []obelisk.BackupDestination
	found := false
	for _, d := range spec.Backup.Destinations {
		if d.ID == id {
			found = true
			continue
		}
		next = append(next, d)
	}
	if !found {
		http.Error(w, "destination not found", http.StatusNotFound)
		return
	}
	spec.Backup.Destinations = next
	obelisk.NormalizeBackup(&spec)
	_ = backup.DeleteDestinationCredentials(cfg, id)
	status, _ := backup.LoadStatus(cfg)
	if status.Destinations != nil {
		delete(status.Destinations, id)
		_ = backup.SaveStatus(cfg, status)
	}
	if err := obelisk.ValidateBackupPolicy(spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}

func (s *Server) handlePromoteBackupDestination(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if obelisk.IsPrimaryBackupDestination(spec, id) {
		writeJSON(w, s.buildBackupConfig(cfg, spec))
		return
	}
	_, promotedFormerID, err := obelisk.PromoteBackupDestination(&spec, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if promotedFormerID != "" && promotedFormerID != "primary" {
		_ = backup.SwapDestinationCredentials(cfg, "primary", promotedFormerID)
		_ = backup.SwapDestinationStatus(cfg, "primary", promotedFormerID)
	}
	if err := obelisk.ValidateBackupPolicy(spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}

func (s *Server) handleTestBackupDestination(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d, ok := obelisk.FindBackupDestination(spec, id)
	if !ok {
		http.Error(w, "destination not found", http.StatusNotFound)
		return
	}
	creds, ok, err := backup.LoadDestinationCredentials(cfg, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		legacy, legOK, err := backup.LoadCredentials(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !legOK {
			http.Error(w, "credentials not configured for destination", http.StatusBadRequest)
			return
		}
		creds = legacy
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	bucket := obelisk.DestinationBucket(spec, d)
	created, testErr := backup.TestRemoteEndpoint(ctx, d.Endpoint, bucket, creds)
	msg := ""
	if testErr != nil {
		msg = testErr.Error()
		_ = backup.UpdateDestinationTest(cfg, id, false, msg)
	} else {
		if created {
			msg = fmt.Sprintf("created bucket %q; write test OK", bucket)
		} else {
			msg = fmt.Sprintf("write test OK (s3://%s/%s)", bucket, obelisk.BackupPreflightObject)
		}
		_ = backup.UpdateDestinationTest(cfg, id, true, msg)
		if b, n, invErr := backup.BucketUsage(ctx, d.Endpoint, bucket, creds, 2000); invErr == nil {
			_ = backup.UpdateDestinationInventory(cfg, id, b, n)
		}
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}

func (s *Server) handleTestAllBackupDestinations(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, d := range obelisk.EnabledBackupDestinations(spec) {
		creds, ok, err := backup.LoadDestinationCredentials(cfg, d.ID)
		if err != nil || !ok {
			if d.ID == "primary" {
				creds, ok, _ = backup.LoadCredentials(cfg)
			}
		}
		if !ok {
			_ = backup.UpdateDestinationTest(cfg, d.ID, false, "credentials not configured")
			continue
		}
		ctx := r.Context()
		bucket := obelisk.DestinationBucket(spec, d)
		_, testErr := backup.TestRemoteEndpoint(ctx, d.Endpoint, bucket, creds)
		if testErr != nil {
			_ = backup.UpdateDestinationTest(cfg, d.ID, false, testErr.Error())
		} else {
			_ = backup.UpdateDestinationTest(cfg, d.ID, true, "write test OK")
			if b, n, invErr := backup.BucketUsage(ctx, d.Endpoint, bucket, creds, 2000); invErr == nil {
				_ = backup.UpdateDestinationInventory(cfg, d.ID, b, n)
			}
		}
	}
	writeJSON(w, s.buildBackupConfig(cfg, spec))
}