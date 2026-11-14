package adminapi

import (
	"encoding/json"
	"net/http"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type instanceView struct {
	obelisk.ServiceInstance
	State string `json:"state"`
}

type createInstanceRequest struct {
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

type patchInstanceRequest struct {
	DisplayName *string                      `json:"display_name,omitempty"`
	Enabled     *bool                        `json:"enabled,omitempty"`
	Credentials *obelisk.InstanceCredentials `json:"credentials,omitempty"`
}

func maskInstance(inst obelisk.ServiceInstance) obelisk.ServiceInstance {
	out := inst
	out.Credentials.Password = ""
	out.Credentials.SecretKey = ""
	if inst.Type == "qdrant" {
		out.Credentials.AccessKey = ""
	}
	return out
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	states := rec.ObserveServiceStates(r.Context(), spec)
	var list []instanceView
	for _, inst := range spec.Instances {
		list = append(list, instanceView{
			ServiceInstance: maskInstance(inst),
			State:           states[inst.StateKey()],
		})
	}
	writeJSON(w, list)
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
	inst, err := obelisk.NewServiceInstance(req.Type, req.DisplayName)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	base := obelisk.SlugID(req.DisplayName)
	if base == "" {
		base = req.Type
	}
	inst.ID = obelisk.UniqueInstanceID(&spec, base)
	spec.Instances = append(spec.Instances, inst)
	obelisk.SyncLegacyServiceFlags(&spec)
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	controller.ReconcileAsync(rec)
	writeJSON(w, map[string]any{
		"instance":       inst,
		"credentials":    inst.Credentials,
		"accepted":       true,
	})
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
	inst, idx, ok := obelisk.FindInstance(&spec, id)
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
	obelisk.SyncLegacyServiceFlags(&spec)
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	controller.ReconcileAsync(rec)
	writeJSON(w, map[string]any{"instance": maskInstance(inst), "accepted": true})
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg := config.EnsureReadablePath(s.ConfigPath)
	s.mu.Lock()
	spec, err := config.Load(cfg)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, idx, ok := obelisk.FindInstance(&spec, id)
	if !ok {
		s.mu.Unlock()
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	inst := spec.Instances[idx]
	inst.Enabled = false
	spec.Instances[idx] = inst
	obelisk.SyncLegacyServiceFlags(&spec)
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	rec := controller.Reconciler{ConfigPath: cfg, HelmRoot: s.HelmRoot}
	controller.ReconcileAsync(rec)
	writeJSON(w, map[string]any{"deleted": id, "accepted": true})
}