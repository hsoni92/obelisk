package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type backupLayoutView struct {
	Bucket       string `json:"bucket"`
	ConfigPrefix string `json:"config_prefix"`
	DataPrefix   string `json:"data_prefix"`
	PreflightKey string `json:"preflight_key"`
}

type backupRemoteView struct {
	Endpoint string `json:"endpoint"`
}

type backupTargetResponse struct {
	MachineID             string           `json:"machine_id"`
	Layout                backupLayoutView `json:"layout"`
	Remote                backupRemoteView `json:"remote"`
	CredentialsSecret     string           `json:"credentials_secret"`
	CredentialsConfigured bool             `json:"credentials_configured"`
	RemoteMetadataReady   bool             `json:"remote_metadata_ready"`
	BackupReady           bool             `json:"backup_ready"`
	LastTestAt            string           `json:"last_test_at,omitempty"`
	LastTestOK            *bool            `json:"last_test_ok,omitempty"`
	LastTestMessage       string           `json:"last_test_message,omitempty"`
	ImplementationStatus  string           `json:"implementation_status"`
}

type patchBackupTargetRequest struct {
	Remote      *backupRemoteView         `json:"remote,omitempty"`
	Credentials *backup.RemoteCredentials `json:"credentials,omitempty"`
}

func (s *Server) handleGetBackupTarget(w http.ResponseWriter, _ *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.buildBackupTarget(cfg, spec))
}

func (s *Server) buildBackupTarget(cfg string, spec obelisk.Spec) backupTargetResponse {
	full := s.buildBackupConfig(cfg, spec)
	credOK, _ := backup.AnyCredentialsConfigured(cfg)
	metaReady := spec.BackupRemoteReady()
	secret := full.CredentialsSecret
	ep := spec.Backup.Remote.Endpoint
	if len(full.Destinations) > 0 {
		ep = full.Destinations[0].Endpoint
	}
	return backupTargetResponse{
		MachineID: full.MachineID,
		Layout:    full.Layout,
		Remote: backupRemoteView{
			Endpoint: ep,
		},
		CredentialsSecret:     secret,
		CredentialsConfigured: credOK,
		RemoteMetadataReady:   metaReady,
		BackupReady:           full.BackupReady,
		ImplementationStatus:  full.ImplementationStatus,
	}
}

func (s *Server) handlePatchBackupTarget(w http.ResponseWriter, r *http.Request) {
	var req patchBackupTargetRequest
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
	obelisk.EnsureMachineIdentity(&spec)
	if req.Remote != nil {
		spec.Backup.Remote.Endpoint = req.Remote.Endpoint
		if len(spec.Backup.Destinations) == 0 {
			spec.Backup.Destinations = []obelisk.BackupDestination{{
				ID: "primary", Label: "Primary", Endpoint: req.Remote.Endpoint,
				Enabled: true, Bucket: spec.BackupBucketName(),
			}}
		} else {
			for i := range spec.Backup.Destinations {
				if spec.Backup.Destinations[i].ID == "primary" {
					spec.Backup.Destinations[i].Endpoint = req.Remote.Endpoint
					break
				}
			}
		}
	}
	if spec.Backup.CredentialsSecret == "" {
		spec.Backup.CredentialsSecret = obelisk.DefaultBackupCredentialsSecret
	}
	if req.Credentials != nil {
		if req.Credentials.AccessKeyID != "" && req.Credentials.SecretAccessKey != "" {
			if err := backup.SaveCredentials(cfg, *req.Credentials); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_ = backup.SaveDestinationCredentials(cfg, "primary", *req.Credentials)
		}
	}
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.buildBackupTarget(cfg, spec))
}

func (s *Server) handleBackupTest(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !spec.BackupRemoteReady() {
		http.Error(w, "backup.remote endpoint and machine identity required", http.StatusBadRequest)
		return
	}
	destID := "primary"
	if dests := obelisk.EnabledBackupDestinations(spec); len(dests) > 0 {
		destID = dests[0].ID
	}
	creds, ok, err := backup.LoadDestinationCredentials(cfg, destID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		creds, ok, err = backup.LoadCredentials(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if !ok {
		http.Error(w, "backup credentials not configured", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	bucketCreated, testErr := backup.TestRemote(ctx, spec, creds)
	resp := s.buildBackupTarget(cfg, spec)
	now := time.Now().UTC().Format(time.RFC3339)
	resp.LastTestAt = now
	if testErr != nil {
		ok := false
		resp.LastTestOK = &ok
		resp.LastTestMessage = testErr.Error()
		_ = backup.UpdateDestinationTest(cfg, destID, false, testErr.Error())
		writeJSON(w, resp)
		return
	}
	okVal := true
	resp.LastTestOK = &okVal
	bucket := spec.BackupBucketName()
	if bucketCreated {
		resp.LastTestMessage = fmt.Sprintf(
			"created bucket %q; write test OK (s3://%s/%s)",
			bucket,
			bucket,
			obelisk.BackupPreflightObject,
		)
	} else {
		resp.LastTestMessage = fmt.Sprintf(
			"write test OK (s3://%s/%s)",
			bucket,
			obelisk.BackupPreflightObject,
		)
	}
	_ = backup.UpdateDestinationTest(cfg, destID, true, resp.LastTestMessage)
	dests := obelisk.EnabledBackupDestinations(spec)
	if len(dests) > 0 {
		d := dests[0]
		bucket = obelisk.DestinationBucket(spec, d)
		if b, n, invErr := backup.BucketUsage(ctx, d.Endpoint, bucket, creds, 2000); invErr == nil {
			_ = backup.UpdateDestinationInventory(cfg, destID, b, n)
		}
	}
	resp.BackupReady = true
	backup.ReloadGlobalSchedule()
	writeJSON(w, resp)
}

func (s *Server) handleBackupRun(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, loadErr := config.Load(cfg)
	if loadErr != nil {
		http.Error(w, loadErr.Error(), http.StatusInternalServerError)
		return
	}
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	states := rec.ObserveServiceStates(r.Context(), spec)
	runner := backup.Runner{ConfigPath: s.ConfigPath, InstanceStates: states}
	ready, msg, err := runner.Ready()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ready {
		http.Error(w, fmt.Sprintf("backup not configured: %s", msg), http.StatusBadRequest)
		return
	}
	out, err := runner.RunNow(r.Context())
	if err != nil {
		if out.UploadedDestinations > 0 {
			msg := err.Error()
			s.recordAudit("Backup", "degraded", msg)
			writeJSON(w, map[string]any{
				"status":     "degraded",
				"message":    msg,
				"uploads":    out.UploadedDestinations,
				"object_key": out.ObjectKey,
			})
			return
		}
		s.recordAudit("Backup", "failed", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	summary := fmt.Sprintf(
		"uploaded %d data object(s) to %d destination(s)",
		len(out.DataObjects),
		out.UploadedDestinations,
	)
	if len(out.DataSkipped) > 0 {
		summary += fmt.Sprintf("; %d skipped", len(out.DataSkipped))
	}
	s.recordAudit("Backup", "completed", summary)
	writeJSON(w, map[string]any{
		"status":       "ok",
		"message":      summary,
		"uploads":      out.UploadedDestinations,
		"object_key":   out.ObjectKey,
		"data_objects": out.DataObjects,
		"data_skipped": out.DataSkipped,
	})
}
