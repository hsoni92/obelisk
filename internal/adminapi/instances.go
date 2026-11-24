package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/backup"
	"github.com/hsoni92/obelisk/internal/backup/lineage"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/credentials"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type instanceDeploymentView struct {
	Release   string                     `json:"release"`
	Namespace string                     `json:"namespace"`
	Backup    obelisk.InstanceBackupView `json:"backup"`
}

type instanceView struct {
	obelisk.ServiceInstance
	State      string                 `json:"state"`
	ReasonCode string                 `json:"reason_code,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
	Origin     string                 `json:"origin,omitempty"` // local | federated
	PeerID     string                 `json:"peer_id,omitempty"`
	Deployment instanceDeploymentView `json:"deployment"`
}

type createInstanceFromBackup struct {
	SourceType       string `json:"source_type"`
	SourceInstanceID string `json:"source_instance_id"`
	BackupID         string `json:"backup_id,omitempty"`
}

type createInstanceRequest struct {
	Type        string                    `json:"type"`
	DisplayName string                    `json:"display_name"`
	FromBackup  *createInstanceFromBackup `json:"from_backup,omitempty"`
}

type patchInstanceBackupRequest struct {
	Enabled             *bool                                 `json:"enabled,omitempty"`
	PolicyOverride      *obelisk.InstanceBackupPolicyOverride `json:"policy_override,omitempty"`
	ClearPolicyOverride *bool                                 `json:"clear_policy_override,omitempty"`
}

type patchInstanceRequest struct {
	Type        *string                      `json:"type,omitempty"`
	DisplayName *string                      `json:"display_name,omitempty"`
	Enabled     *bool                        `json:"enabled,omitempty"`
	Backup      *patchInstanceBackupRequest  `json:"backup,omitempty"`
	Credentials *obelisk.InstanceCredentials `json:"credentials,omitempty"`
}

func instanceNamespace() string {
	if ns := strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE")); ns != "" {
		return ns
	}
	return "obelisk"
}

func enrichInstanceView(inst obelisk.ServiceInstance, state string, spec obelisk.Spec) instanceView {
	global := obelisk.ResolvedBackupPolicy(spec)
	view := instanceView{
		ServiceInstance: maskInstance(inst),
		State:           state,
		Deployment: instanceDeploymentView{
			Release:   inst.ReleaseName(),
			Namespace: instanceNamespace(),
			Backup:    obelisk.BuildInstanceBackupView(inst, global),
		},
	}
	if e := controller.GlobalEngine(); e != nil && e.Scheduler != nil {
		if st := e.Scheduler.InstanceStatus(inst.StateKey()); st != nil {
			if st.State != "" {
				view.State = st.State
			}
			view.ReasonCode = st.ReasonCode
			view.Reason = st.Detail
		}
	}
	return view
}

func maskInstance(inst obelisk.ServiceInstance) obelisk.ServiceInstance {
	out := inst
	out.Credentials.Password = ""
	out.Credentials.SecretKey = ""
	out.Credentials.AccessKey = ""
	return out
}

// listInstanceViews merges local yaml instances with federated peer catalog on gateways.
func listInstanceViews(cfg string, spec obelisk.Spec, states map[string]string) []instanceView {
	var list []instanceView
	for _, inst := range spec.Instances {
		v := enrichInstanceView(inst, states[inst.StateKey()], spec)
		v.Origin = "local"
		list = append(list, v)
	}
	if !obelisk.IsGateway(spec) {
		return list
	}
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		return list
	}
	for _, ft := range reg.FederatedTargets(spec, 0) {
		st := "running"
		reason := ""
		if !ft.Reachable {
			st = "degraded"
			reason = "peer_unreachable"
		}
		v := enrichInstanceView(ft.Inst, st, spec)
		v.Origin = "federated"
		v.PeerID = ft.PeerID
		v.ReasonCode = reason
		v.Reason = reason
		list = append(list, v)
	}
	return list
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rec := s.reconciler(cfg)
	states := rec.ObserveServiceStates(r.Context(), spec)
	writeJSON(w, listInstanceViews(cfg, spec, states))
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	var req createInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var (
		inst       obelisk.ServiceInstance
		sourceNode lineage.Node
		sourceBkp  string
		restoreOK  bool
		restoreErr string
	)
	if req.FromBackup != nil {
		fb := req.FromBackup
		srcType := strings.ToLower(strings.TrimSpace(fb.SourceType))
		srcID := strings.TrimSpace(fb.SourceInstanceID)
		reqType := strings.ToLower(strings.TrimSpace(req.Type))
		if srcType == "" || srcID == "" {
			s.mu.Unlock()
			http.Error(w, "from_backup requires source_type and source_instance_id", http.StatusBadRequest)
			return
		}
		if reqType != srcType {
			s.mu.Unlock()
			http.Error(w, "instance type must match backup source type", http.StatusBadRequest)
			return
		}
		if !obelisk.ServiceTypeSupportsBackup(reqType) {
			s.mu.Unlock()
			http.Error(w, "this service type does not support data backups", http.StatusBadRequest)
			return
		}
		inst, sourceNode, err = backup.PrepareCloneInstance(cfg, &spec, srcType, srcID, fb.BackupID, "", req.DisplayName)
		if err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sourceBkp = sourceNode.ID
	} else {
		id := obelisk.AllocateInstanceID(&spec)
		inst, err = obelisk.NewServiceInstance(req.Type, id, req.DisplayName)
		if err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		spec.Instances = append(spec.Instances, inst)
	}
	if err := obelisk.ValidateInstanceBackupPolicyOverride(spec, inst); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	obelisk.SyncLegacyServiceFlags(&spec)
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	s.recordAudit("Instance", "created", inst.Type+"/"+inst.ID)
	if eng := controller.GlobalEngine(); eng != nil {
		eng.TriggerInstance(inst.StateKey())
	} else {
		controller.ReconcileAsync(s.reconciler(cfg))
	}

	if req.FromBackup != nil {
		ctx := r.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		out, _ := backup.CompleteCloneRestore(ctx, cfg, s.HelmRoot, spec, inst, sourceNode, sourceBkp)
		restoreOK = out.RestoreOK
		restoreErr = out.RestoreError
	}

	ns := instanceNamespace()
	resolved := obelisk.SpecWithResolvedCredentials(spec, func(in obelisk.ServiceInstance) obelisk.InstanceCredentials {
		return credentials.MergeInstanceCredentials(r.Context(), ns, in)
	})
	for i := range spec.Instances {
		if spec.Instances[i].Type == inst.Type && spec.Instances[i].ID == inst.ID {
			inst = spec.Instances[i]
			break
		}
	}
	use := ""
	for _, c := range obelisk.BuildConnections(resolved, nil).Instances {
		if c.ID == inst.ID && c.Type == inst.Type {
			use = c.Use
			break
		}
	}
	resp := map[string]any{
		"instance":    inst,
		"credentials": inst.Credentials,
		"use":         use,
		"accepted":    true,
	}
	if req.FromBackup != nil {
		resp["from_backup"] = map[string]any{
			"source_type":        strings.ToLower(strings.TrimSpace(req.FromBackup.SourceType)),
			"source_instance_id": strings.TrimSpace(req.FromBackup.SourceInstanceID),
			"backup_id":          sourceBkp,
			"restore_ok":         restoreOK,
			"restore_error":      restoreErr,
		}
	}
	writeJSON(w, resp)
}

func (s *Server) handlePatchInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req patchInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var inst obelisk.ServiceInstance
	var idx int
	var ok bool
	if req.Type != nil && strings.TrimSpace(*req.Type) != "" {
		inst, idx, ok = obelisk.FindInstanceByTypeAndID(&spec, *req.Type, id)
	} else {
		inst, idx, ok = obelisk.FindInstance(&spec, id)
	}
	if !ok {
		s.mu.Unlock()
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	if req.DisplayName != nil {
		inst.DisplayName = *req.DisplayName
	}
	if req.Enabled != nil {
		inst.Enabled = *req.Enabled
	}
	if req.Backup != nil {
		if req.Backup.Enabled != nil {
			inst.Backup.Enabled = req.Backup.Enabled
		}
		if req.Backup.ClearPolicyOverride != nil && *req.Backup.ClearPolicyOverride {
			inst.Backup.PolicyOverride = nil
		} else if req.Backup.PolicyOverride != nil {
			if inst.Backup.PolicyOverride == nil {
				inst.Backup.PolicyOverride = &obelisk.InstanceBackupPolicyOverride{}
			}
			po := inst.Backup.PolicyOverride
			if req.Backup.PolicyOverride.Schedule != nil {
				s := strings.TrimSpace(*req.Backup.PolicyOverride.Schedule)
				if s != "" {
					if err := backup.ValidateCronSchedule(s); err != nil {
						http.Error(w, "invalid backup schedule cron", http.StatusBadRequest)
						return
					}
				}
				po.Schedule = &s
			}
			if req.Backup.PolicyOverride.ReplicationFactor != nil {
				po.ReplicationFactor = req.Backup.PolicyOverride.ReplicationFactor
			}
			if req.Backup.PolicyOverride.DataRetentionDays != nil {
				po.DataRetentionDays = req.Backup.PolicyOverride.DataRetentionDays
			}
		}
	}
	if req.Credentials != nil {
		c := *req.Credentials
		if c.Username != "" {
			inst.Credentials.Username = c.Username
		}
		if c.Password != "" {
			inst.Credentials.Password = c.Password
		}
		if c.Database != "" {
			inst.Credentials.Database = c.Database
		}
		if c.AccessKey != "" {
			inst.Credentials.AccessKey = c.AccessKey
		}
		if c.SecretKey != "" {
			inst.Credentials.SecretKey = c.SecretKey
		}
	}
	spec.Instances[idx] = inst
	if err := obelisk.ValidateInstanceBackupPolicyOverride(spec, inst); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	obelisk.SyncLegacyServiceFlags(&spec)
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	s.recordAudit("Instance", "updated", inst.Type+"/"+inst.ID)
	if eng := controller.GlobalEngine(); eng != nil {
		eng.TriggerInstance(inst.StateKey())
	} else {
		controller.ReconcileAsync(s.reconciler(cfg))
	}
	interim := "starting"
	if req.Enabled != nil && !*req.Enabled {
		interim = "stopping"
	}
	writeJSON(w, map[string]any{
		"instance": enrichInstanceView(inst, interim, spec),
		"accepted": true,
	})
}

func lookupInstance(spec *obelisk.Spec, typ, id string) (obelisk.ServiceInstance, bool) {
	if strings.TrimSpace(typ) != "" {
		inst, _, ok := obelisk.FindInstanceByTypeAndID(spec, typ, id)
		return inst, ok
	}
	inst, _, ok := obelisk.FindInstance(spec, id)
	return inst, ok
}

func (s *Server) handlePostInstanceRestart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	typ := strings.TrimSpace(r.URL.Query().Get("type"))
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	inst, ok := lookupInstance(&spec, typ, id)
	if !ok {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	rec := s.reconciler(cfg)
	out := rec.RestartInstanceWorkload(ctx, inst)
	detail := inst.StateKey() + ": " + out.Message
	if out.Method == "skipped" {
		writeJSON(w, map[string]any{
			"status":    "skipped",
			"method":    out.Method,
			"message":   out.Message,
			"state_key": inst.StateKey(),
		})
		return
	}
	s.recordAudit("Instance", "restart", detail)
	writeJSON(w, map[string]any{
		"status":    "accepted",
		"method":    out.Method,
		"message":   out.Message,
		"state_key": inst.StateKey(),
	})
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	typ := strings.TrimSpace(r.URL.Query().Get("type"))
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var inst obelisk.ServiceInstance
	var ok bool
	if typ != "" {
		inst, ok = obelisk.RemoveInstanceByTypeAndID(&spec, typ, id)
	} else {
		var idx int
		inst, idx, ok = obelisk.FindInstance(&spec, id)
		if ok {
			spec.Instances = append(spec.Instances[:idx], spec.Instances[idx+1:]...)
		}
	}
	if !ok {
		s.mu.Unlock()
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	obelisk.SyncLegacyServiceFlags(&spec)
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	deleteBackups := strings.EqualFold(r.URL.Query().Get("delete_backups"), "true")
	instKey := inst.Type + "/" + inst.ID
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if deleteBackups {
		idx, loadErr := lineage.Load(cfg)
		if loadErr == nil {
			if nodes, forest, ok := lineage.ListNodes(idx, inst.Type, inst.ID); ok {
				var keys []string
				for _, n := range nodes {
					keys = append(keys, n.ObjectKey, n.ManifestKey)
				}
				_ = forest
				_ = backup.DeleteObjectsAllDestinations(ctx, cfg, spec, keys)
			}
		}
		_ = lineage.RemoveForest(cfg, inst.Type, inst.ID)
	} else {
		_ = lineage.MarkOrphan(cfg, inst.Type, inst.ID)
	}
	s.recordAudit("Instance", "terminated", instKey)
	controller.MarkReleaseDeleteIntent(inst.ReleaseName())
	controller.ReconcileAsync(s.reconciler(cfg))
	writeJSON(w, map[string]any{
		"deleted":          instKey,
		"release":          inst.ReleaseName(),
		"accepted":         true,
		"backups_deleted":  deleteBackups,
		"backups_orphaned": !deleteBackups,
	})
}
