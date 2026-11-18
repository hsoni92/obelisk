package adminapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/credentials"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type instanceDatabaseView struct {
	BootstrapDatabase string                        `json:"bootstrap_database"`
	Additional        []string                      `json:"additional_databases"`
	Connections       []obelisk.DatabaseConnection  `json:"connections"`
}

func (s *Server) handleGetInstanceDatabases(w http.ResponseWriter, r *http.Request) {
	inst, spec, ok := s.loadRelationalInstance(w, r)
	if !ok {
		return
	}
	ns := instanceNamespace()
	resolved := obelisk.SpecWithResolvedCredentials(spec, func(in obelisk.ServiceInstance) obelisk.InstanceCredentials {
		return credentials.MergeInstanceCredentials(r.Context(), ns, in)
	})
	writeJSON(w, instanceDatabaseResponse(spec, resolved, inst))
}

func (s *Server) handlePostInstanceDatabase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(strings.ToLower(body.Name))
	if err := obelisk.ValidateDatabaseName(name); err != nil {
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
	inst, idx, ok := findInstanceInRequest(r, &spec)
	if !ok || !obelisk.IsRelationalEngine(inst.Type) {
		s.mu.Unlock()
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	for _, existing := range inst.Databases {
		if strings.EqualFold(existing, name) {
			s.mu.Unlock()
			http.Error(w, "database already registered", http.StatusConflict)
			return
		}
	}
	inst.Databases = append(inst.Databases, name)
	spec.Instances[idx] = inst
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	s.recordAudit("InstanceDatabase", "created", inst.Type+"/"+inst.ID+"/"+name)
	controller.ReconcileAsync(s.reconciler(cfg))
	writeJSON(w, map[string]any{
		"accepted": true,
		"database": instanceDatabaseResponse(spec, spec, inst),
	})
}

func (s *Server) handleDeleteInstanceDatabase(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(strings.ToLower(r.PathValue("dbname")))
	if dbName == "" {
		http.Error(w, "database name required", http.StatusBadRequest)
		return
	}
	if dbName == obelisk.BootstrapDatabaseName {
		http.Error(w, "cannot remove bootstrap database", http.StatusBadRequest)
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
	inst, idx, ok := findInstanceInRequest(r, &spec)
	if !ok || !obelisk.IsRelationalEngine(inst.Type) {
		s.mu.Unlock()
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}
	var kept []string
	var found bool
	for _, name := range inst.Databases {
		if strings.EqualFold(name, dbName) {
			found = true
			continue
		}
		kept = append(kept, name)
	}
	if !found {
		s.mu.Unlock()
		http.Error(w, "database not registered", http.StatusNotFound)
		return
	}
	inst.Databases = kept
	spec.Instances[idx] = inst
	if err := config.Save(cfg, spec); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Unlock()
	s.recordAudit("InstanceDatabase", "deleted", inst.Type+"/"+inst.ID+"/"+dbName)
	controller.ReconcileAsync(s.reconciler(cfg))
	writeJSON(w, map[string]any{
		"accepted": true,
		"database": instanceDatabaseResponse(spec, spec, inst),
	})
}

func findInstanceInRequest(r *http.Request, spec *obelisk.Spec) (obelisk.ServiceInstance, int, bool) {
	id := r.PathValue("id")
	if typ := strings.TrimSpace(r.URL.Query().Get("type")); typ != "" {
		return obelisk.FindInstanceByTypeAndID(spec, typ, id)
	}
	return obelisk.FindInstance(spec, id)
}

func (s *Server) loadRelationalInstance(w http.ResponseWriter, r *http.Request) (obelisk.ServiceInstance, obelisk.Spec, bool) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return obelisk.ServiceInstance{}, obelisk.Spec{}, false
	}
	inst, _, ok := findInstanceInRequest(r, &spec)
	if !ok || !obelisk.IsRelationalEngine(inst.Type) {
		http.Error(w, "relational instance not found", http.StatusNotFound)
		return obelisk.ServiceInstance{}, obelisk.Spec{}, false
	}
	return inst, spec, true
}

func instanceDatabaseResponse(spec obelisk.Spec, credSpec obelisk.Spec, inst obelisk.ServiceInstance) instanceDatabaseView {
	// locate instance with credentials in credSpec
	resolvedInst := inst
	for _, in := range credSpec.Instances {
		if in.Type == inst.Type && in.ID == inst.ID {
			resolvedInst = in
			break
		}
	}
	conns := []obelisk.DatabaseConnection{}
	for _, name := range obelisk.DesiredDatabaseNames(resolvedInst) {
		if u := obelisk.RelationalConnectionURL(spec, resolvedInst, name); u != "" {
			conns = append(conns, obelisk.DatabaseConnection{Name: name, URL: u})
		}
	}
	return instanceDatabaseView{
		BootstrapDatabase: obelisk.BootstrapDatabase(resolvedInst),
		Additional:        append([]string(nil), resolvedInst.Databases...),
		Connections:       conns,
	}
}