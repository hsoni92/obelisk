package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type backupLayoutView struct {
	Bucket        string `json:"bucket"`
	ConfigPrefix  string `json:"config_prefix"`
	DataPrefix    string `json:"data_prefix"`
	PreflightKey  string `json:"preflight_key"`
}

type backupRemoteView struct {
	Endpoint string `json:"endpoint"`
}

type backupTargetResponse struct {
	MachineID               string           `json:"machine_id"`
	Layout                  backupLayoutView `json:"layout"`
	Remote                  backupRemoteView `json:"remote"`
	CredentialsSecret       string           `json:"credentials_secret"`
	CredentialsConfigured   bool             `json:"credentials_configured"`
	RemoteMetadataReady     bool             `json:"remote_metadata_ready"`
	BackupReady             bool             `json:"backup_ready"`
	LastTestAt              string           `json:"last_test_at,omitempty"`
	LastTestOK              *bool            `json:"last_test_ok,omitempty"`
	LastTestMessage         string           `json:"last_test_message,omitempty"`
	ImplementationStatus    string           `json:"implementation_status"`
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
	credOK, _ := backup.CredentialsConfigured(cfg)
	metaReady := spec.BackupRemoteReady()
	secret := spec.Backup.CredentialsSecret
	if secret == "" {
		secret = obelisk.DefaultBackupCredentialsSecret
	}
	ready := metaReady && credOK
	bucket := spec.BackupBucketName()
	return backupTargetResponse{
		MachineID: bucket,
		Layout: backupLayoutView{
			Bucket:       bucket,
			ConfigPrefix: obelisk.BackupConfigPrefix,
			DataPrefix:   obelisk.BackupDataPrefix,
			PreflightKey: obelisk.BackupPreflightObject,
		},
		Remote: backupRemoteView{
			Endpoint: spec.Backup.Remote.Endpoint,
		},
		CredentialsSecret:     secret,
		CredentialsConfigured: credOK,
		RemoteMetadataReady:   metaReady,
		BackupReady:           ready,
		ImplementationStatus:  "stub",
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
	creds, ok, err := backup.LoadCredentials(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "backup credentials not configured", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	testErr := backup.TestRemote(ctx, spec, creds)
	resp := s.buildBackupTarget(cfg, spec)
	now := time.Now().UTC().Format(time.RFC3339)
	resp.LastTestAt = now
	if testErr != nil {
		ok := false
		resp.LastTestOK = &ok
		resp.LastTestMessage = testErr.Error()
		writeJSON(w, resp)
		return
	}
	okVal := true
	resp.LastTestOK = &okVal
	resp.LastTestMessage = fmt.Sprintf(
		"write test OK (s3://%s/%s)",
		spec.BackupBucketName(),
		obelisk.BackupPreflightObject,
	)
	resp.BackupReady = true
	writeJSON(w, resp)
}

func (s *Server) handleBackupRun(w http.ResponseWriter, r *http.Request) {
	runner := backup.Runner{ConfigPath: s.ConfigPath}
	ready, msg, err := runner.Ready()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ready {
		http.Error(w, fmt.Sprintf("backup not configured: %s", msg), http.StatusBadRequest)
		return
	}
	if err := runner.RunNow(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{
		"status":  "accepted",
		"message": "backup preflight passed; upload worker still stub in v1",
	})
}