package adminapi

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/credentials"
	"github.com/hsoni92/obelisk/internal/history"
	"github.com/hsoni92/obelisk/internal/mailgateway"
	"github.com/hsoni92/obelisk/internal/openroutergateway"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/internal/restore"
	"github.com/hsoni92/obelisk/internal/vitals"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type Server struct {
	ConfigPath string
	HelmRoot   string
	Gateway    *openroutergateway.Server
	// GatewayListenAddr is the controller's OpenRouter listen socket (e.g. ":18081").
	GatewayListenAddr     string
	MailGateway           *mailgateway.Server
	MailGatewayListenAddr string
	Blueprint             *blueprint.Runtime
	Restore               *restore.Orchestrator
	mu                    sync.Mutex
	statesCache           statesCache
	historyOnce           sync.Once
	historyStore          *history.Store
	Vitals                *vitals.Store
	updateCache           updateCache
}

func (s *Server) reconciler(cfg string) controller.Reconciler {
	ns := strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE"))
	if ns == "" {
		ns = "obelisk"
	}
	return controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot, Namespace: ns}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tailscale/status", s.handleTailscaleStatus)
	mux.HandleFunc("POST /api/v1/tailscale/connect", s.handleTailscaleConnect)
	mux.HandleFunc("PATCH /api/v1/deployment", s.handlePatchDeployment)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/system/health", s.handleSystemHealth)
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	mux.HandleFunc("GET /api/v1/update", s.handleUpdateStatus)
	mux.HandleFunc("GET /api/v1/overview", s.handleOverview)
	mux.HandleFunc("GET /api/v1/activity/history", s.handleActivityHistory)
	mux.HandleFunc("GET /api/v1/connections", s.handleConnections)
	mux.HandleFunc("POST /api/v1/diagnostics/run", s.handleDiagnosticsRun)
	mux.HandleFunc("GET /api/v1/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/v1/config", s.handlePutConfig)
	mux.HandleFunc("POST /api/v1/services/{name}/enable", s.handleServiceEnable)
	mux.HandleFunc("POST /api/v1/services/{name}/disable", s.handleServiceDisable)
	mux.HandleFunc("POST /api/v1/peers/advertise", s.handlePeerAdvertise)
	mux.HandleFunc("POST /api/v1/peers/resign", s.handlePeerResign)
	mux.HandleFunc("GET /api/v1/peers", s.handleListPeers)
	mux.HandleFunc("POST /api/v1/peers/{id}/admit", s.handleAdmitPeer)
	mux.HandleFunc("DELETE /api/v1/peers/{id}", s.handleDeletePeer)
	mux.HandleFunc("GET /api/v1/federation", s.handleGetFederation)
	mux.HandleFunc("PATCH /api/v1/federation", s.handlePatchFederation)
	mux.HandleFunc("POST /api/v1/federation/advertise", s.handleFederationAdvertiseNow)
	mux.HandleFunc("POST /api/v1/federation/unregister", s.handleFederationUnregister)
	mux.HandleFunc("GET /api/v1/instances", s.handleListInstances)
	mux.HandleFunc("POST /api/v1/instances", s.handleCreateInstance)
	mux.HandleFunc("PATCH /api/v1/instances/{id}", s.handlePatchInstance)
	mux.HandleFunc("DELETE /api/v1/instances/{id}", s.handleDeleteInstance)
	mux.HandleFunc("POST /api/v1/instances/{id}/restart", s.handlePostInstanceRestart)
	mux.HandleFunc("GET /api/v1/instances/{id}/databases", s.handleGetInstanceDatabases)
	mux.HandleFunc("POST /api/v1/instances/{id}/databases", s.handlePostInstanceDatabase)
	mux.HandleFunc("DELETE /api/v1/instances/{id}/databases/{dbname}", s.handleDeleteInstanceDatabase)
	mux.HandleFunc("GET /api/v1/backup/target", s.handleGetBackupTarget)
	mux.HandleFunc("PATCH /api/v1/backup/target", s.handlePatchBackupTarget)
	mux.HandleFunc("GET /api/v1/backup/config", s.handleGetBackupConfig)
	mux.HandleFunc("PATCH /api/v1/backup/policy", s.handlePatchBackupPolicy)
	mux.HandleFunc("POST /api/v1/backup/destinations", s.handlePostBackupDestination)
	mux.HandleFunc("PATCH /api/v1/backup/destinations/{id}", s.handlePatchBackupDestination)
	mux.HandleFunc("DELETE /api/v1/backup/destinations/{id}", s.handleDeleteBackupDestination)
	mux.HandleFunc("POST /api/v1/backup/destinations/{id}/promote-primary", s.handlePromoteBackupDestination)
	mux.HandleFunc("POST /api/v1/backup/destinations/{id}/test", s.handleTestBackupDestination)
	mux.HandleFunc("POST /api/v1/backup/destinations/test-all", s.handleTestAllBackupDestinations)
	mux.HandleFunc("POST /api/v1/backup/test", s.handleBackupTest)
	mux.HandleFunc("POST /api/v1/backup/run", s.handleBackupRun)
	mux.HandleFunc("POST /api/v1/backups/clone", s.handlePostCloneBackup)
	mux.HandleFunc("GET /api/v1/backup/catalog", s.handleGetBackupCatalog)
	mux.HandleFunc("POST /api/v1/backup/index/rebuild", s.handlePostBackupIndexRebuild)
	mux.HandleFunc("GET /api/v1/instances/{id}/backups", s.handleGetInstanceBackups)
	mux.HandleFunc("POST /api/v1/instances/{id}/backup/run", s.handlePostInstanceBackupRun)
	mux.HandleFunc("POST /api/v1/instances/{id}/backup/restore-active", s.handlePostInstanceRestoreActive)
	mux.HandleFunc("DELETE /api/v1/instances/{id}/backups/{backupId}", s.handleDeleteInstanceBackup)
	mux.HandleFunc("DELETE /api/v1/instances/{id}/backups", s.handleDeleteInstanceBackupForest)
	mux.HandleFunc("GET /discover/v1/projects/{id}", s.handleDiscoverProject)
	mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/v1/projects", s.handleCreateProject)
	mux.HandleFunc("PATCH /api/v1/projects/{id}", s.handlePatchProject)
	mux.HandleFunc("DELETE /api/v1/projects/{id}", s.handleDeleteProject)
	mux.HandleFunc("POST /api/v1/projects/{id}/rotate-discovery-token", s.handleRotateDiscoveryToken)
	mux.HandleFunc("GET /api/v1/openrouter/config", s.handleGetOpenRouterConfig)
	mux.HandleFunc("PATCH /api/v1/openrouter/config", s.handlePatchOpenRouterConfig)
	mux.HandleFunc("POST /api/v1/openrouter/accounts", s.handlePostOpenRouterAccount)
	mux.HandleFunc("PATCH /api/v1/openrouter/accounts/{id}", s.handlePatchOpenRouterAccount)
	mux.HandleFunc("DELETE /api/v1/openrouter/accounts/{id}", s.handleDeleteOpenRouterAccount)
	mux.HandleFunc("POST /api/v1/openrouter/test", s.handleOpenRouterTest)
	mux.HandleFunc("GET /api/v1/mail/config", s.handleGetMailConfig)
	mux.HandleFunc("PATCH /api/v1/mail/config", s.handlePatchMailConfig)
	mux.HandleFunc("POST /api/v1/mail/accounts", s.handlePostMailAccount)
	mux.HandleFunc("PATCH /api/v1/mail/accounts/{id}", s.handlePatchMailAccount)
	mux.HandleFunc("DELETE /api/v1/mail/accounts/{id}", s.handleDeleteMailAccount)
	mux.HandleFunc("POST /api/v1/mail/test", s.handleMailTest)
	mux.HandleFunc("GET /api/v1/blueprint/status", s.handleBlueprintStatus)
	mux.HandleFunc("GET /api/v1/blueprint/github/config", s.handleBlueprintGitHubConfig)
	mux.HandleFunc("POST /api/v1/blueprint/github/authorize", s.handleBlueprintAuthorize)
	mux.HandleFunc("POST /api/v1/blueprint/github/callback", s.handleBlueprintCallback)
	mux.HandleFunc("POST /api/v1/blueprint/github/logout", s.handleBlueprintLogout)
	mux.HandleFunc("POST /api/v1/blueprint/onboard", s.handleBlueprintOnboard)
	mux.HandleFunc("POST /api/v1/blueprint/unlock", s.handleBlueprintUnlock)
	mux.HandleFunc("POST /api/v1/blueprint/lock", s.handleBlueprintLock)
	mux.HandleFunc("POST /api/v1/blueprint/import", s.handleBlueprintImport)
	mux.HandleFunc("POST /api/v1/blueprint/push", s.handleBlueprintPush)
	mux.HandleFunc("POST /api/v1/restore/chain", s.handleRestoreChain)
	mux.HandleFunc("GET /api/v1/restore/chain", s.handleRestoreChainStatus)
	return normalizeLegacyAPIPath(s.requireGitHub(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	rec := s.reconciler(cfg)
	states := rec.ObserveServiceStates(r.Context(), spec)

	reconcileMode := "cluster"
	if rec.SimulateMode() {
		reconcileMode = "off"
	}

	instances := listInstanceViews(cfg, spec, states)
	bt := s.buildBackupTarget(cfg, spec)
	writeJSON(w, map[string]any{
		"machineId":     spec.BackupBucketName(),
		"profile":       spec.Deployment.Profile,
		"role":          spec.Deployment.Role,
		"tailscale":     spec.Deployment.Tailscale.Enabled,
		"configPath":    cfg,
		"reconcileMode": reconcileMode,
		"serviceStates": states,
		"instances":     instances,
		"messages":      []string{},
		"backup": map[string]any{
			"ready":                  bt.BackupReady,
			"credentials_configured": bt.CredentialsConfigured,
			"remote_metadata_ready":  bt.RemoteMetadataReady,
			"implementation_status":  bt.ImplementationStatus,
			"bucket":                 bt.Layout.Bucket,
			"config_prefix":          bt.Layout.ConfigPrefix,
			"data_prefix":            bt.Layout.DataPrefix,
		},
	})
}

func desiredServiceStates(spec obelisk.Spec) map[string]string {
	states := map[string]string{
		"postgres": "stopped",
		"redis":    "stopped",
		"rustfs":   "stopped",
	}
	if spec.Services.Postgres.Enabled {
		states["postgres"] = "pending"
	}
	if spec.Services.Redis.Enabled {
		states["redis"] = "pending"
	}
	if spec.Services.RustFS.Enabled {
		states["rustfs"] = "pending"
	}
	return states
}

func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	spec, err := config.Load(config.EnsureReadablePath(s.ConfigPath))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, spec)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var spec obelisk.Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := config.Save(s.ConfigPath, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.recordAudit("Config", "updated", "Cluster configuration saved")
	writeJSON(w, map[string]string{"status": "saved"})
}

func (s *Server) handleServiceEnable(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := controller.SetServiceEnabled(&spec, name, true); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()

	s.recordAudit("Service type", "enabled", name)
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	controller.ReconcileAsync(rec)
	states := rec.ObserveServiceStates(r.Context(), spec)
	writeJSON(w, map[string]any{
		"enabled":       name,
		"accepted":      true,
		"serviceStates": states,
	})
}

func (s *Server) handleServiceDisable(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := controller.SetServiceEnabled(&spec, name, false); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()

	s.recordAudit("Service type", "disabled", name)
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	controller.ReconcileAsync(rec)
	states := rec.ObserveServiceStates(r.Context(), spec)
	writeJSON(w, map[string]any{"disabled": name, "accepted": true, "serviceStates": states})
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	spec, err := config.Load(config.EnsureReadablePath(s.ConfigPath))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	rec := s.reconciler(cfg)
	states := rec.ObserveServiceStates(r.Context(), spec)
	ns := rec.Namespace
	if ns == "" {
		ns = instanceNamespace()
	}
	resolved := obelisk.SpecWithResolvedCredentials(spec, func(inst obelisk.ServiceInstance) obelisk.InstanceCredentials {
		return credentials.MergeInstanceCredentials(r.Context(), ns, inst)
	})
	conns := obelisk.BuildConnections(resolved, states)
	if obelisk.IsGateway(spec) {
		if reg, err := peer.LoadRegistry(cfg); err == nil {
			conns = peer.AppendFederatedConnections(spec, reg, conns)
		}
	}
	writeJSON(w, conns)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
