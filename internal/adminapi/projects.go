package adminapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/credentials"
	disc "github.com/hsoni92/obelisk/internal/discovery"
	"github.com/hsoni92/obelisk/internal/mail"
	"github.com/hsoni92/obelisk/internal/openrouter"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type projectView struct {
	ID                        string   `json:"id"`
	Name                      string   `json:"name"`
	InstanceIDs               []string `json:"instance_ids,omitempty"`
	DiscoveryEnabled          bool     `json:"discovery_enabled"`
	DiscoveryConfigured       bool     `json:"discovery_configured"`
	DiscoveryURL              string   `json:"discovery_url,omitempty"`
	DiscoveryURLInCluster     string   `json:"discovery_url_in_cluster,omitempty"`
	DiscoveryKongInClusterURL string   `json:"discovery_kong_in_cluster_url,omitempty"`
}

func (s *Server) enrichProjectView(cfg string, p obelisk.Project) projectView {
	v := projectView{
		ID:               p.ID,
		Name:             p.Name,
		InstanceIDs:      append([]string(nil), p.InstanceIDs...),
		DiscoveryEnabled: p.DiscoveryEnabled,
	}
	if p.DiscoveryEnabled {
		v.DiscoveryURL = obelisk.ProjectDiscoveryPath(p.ID)
		v.DiscoveryURLInCluster = obelisk.ProjectDiscoveryPathInCluster(p.ID)
		v.DiscoveryKongInClusterURL = obelisk.ProjectDiscoveryKongInClusterURL(p.ID, instanceNamespace())
		tok := disc.ReadToken(cfg, p.ID)
		v.DiscoveryConfigured = strings.TrimSpace(tok) != ""
	}
	return v
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	spec, err := config.Load(config.EnsureReadablePath(s.ConfigPath))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cfg := config.EnsureReadablePath(s.ConfigPath)
	var out []projectView
	for _, p := range spec.Projects {
		out = append(out, s.enrichProjectView(cfg, p))
	}
	writeJSON(w, out)
}

type createProjectRequest struct {
	Name             string   `json:"name"`
	InstanceIDs      []string `json:"instance_ids"`
	DiscoveryEnabled *bool    `json:"discovery_enabled"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := obelisk.Project{
		ID:          obelisk.NewProjectID(),
		Name:        name,
		InstanceIDs: req.InstanceIDs,
	}
	if req.DiscoveryEnabled != nil {
		p.DiscoveryEnabled = *req.DiscoveryEnabled
	}
	if err := obelisk.ValidateProjectInstanceRefs(spec, p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, existing := range spec.Projects {
		if strings.EqualFold(existing.Name, p.Name) {
			http.Error(w, "project name already exists", http.StatusBadRequest)
			return
		}
	}
	spec.Projects = append(spec.Projects, p)
	spec.Projects = obelisk.NormalizeProjects(spec.Projects)
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.recordAudit("Project", "created", p.Name)
	writeJSON(w, s.enrichProjectView(cfg, p))
}

type patchProjectRequest struct {
	Name             *string  `json:"name"`
	InstanceIDs      []string `json:"instance_ids"`
	DiscoveryEnabled *bool    `json:"discovery_enabled"`
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	var req patchProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := obelisk.FindProject(spec, id)
	if p == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			http.Error(w, "name cannot be empty", http.StatusBadRequest)
			return
		}
		for _, existing := range spec.Projects {
			if existing.ID != id && strings.EqualFold(existing.Name, n) {
				http.Error(w, "project name already exists", http.StatusBadRequest)
				return
			}
		}
		p.Name = n
	}
	if req.InstanceIDs != nil {
		p.InstanceIDs = req.InstanceIDs
	}
	if req.DiscoveryEnabled != nil {
		p.DiscoveryEnabled = *req.DiscoveryEnabled
	}
	if err := obelisk.ValidateProjectInstanceRefs(spec, *p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	spec.Projects = obelisk.NormalizeProjects(spec.Projects)
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !p.DiscoveryEnabled {
		disc.DeleteToken(cfg, p.ID)
	}
	s.recordAudit("Project", "updated", p.Name)
	writeJSON(w, s.enrichProjectView(cfg, *p))
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var kept []obelisk.Project
	var removed *obelisk.Project
	for _, p := range spec.Projects {
		if p.ID == id {
			cp := p
			removed = &cp
			continue
		}
		kept = append(kept, p)
	}
	if removed == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	spec.Projects = kept
	if err := config.Save(cfg, spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	disc.DeleteToken(cfg, id)
	s.recordAudit("Project", "deleted", removed.Name)
	writeJSON(w, map[string]string{"deleted": id})
}

type rotateDiscoveryTokenResponse struct {
	Token string `json:"token"`
}

func (s *Server) handleRotateDiscoveryToken(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := obelisk.FindProject(spec, id)
	if p == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	if !p.DiscoveryEnabled {
		http.Error(w, "discovery not enabled for project", http.StatusBadRequest)
		return
	}
	tok, err := newDiscoveryToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := disc.WriteToken(cfg, id, tok); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.recordAudit("Project", "discovery_token_rotated", p.Name)
	writeJSON(w, rotateDiscoveryTokenResponse{Token: tok})
}

func projectDiscoveryExtras(cfg string) obelisk.ProjectDiscoveryExtras {
	var extras obelisk.ProjectDiscoveryExtras
	if orc, ok, _ := openrouter.LoadCredentials(cfg); ok {
		extras.OpenRouterGatewayToken = strings.TrimSpace(orc.GatewayToken)
	}
	if mc, ok, _ := mail.LoadCredentials(cfg); ok {
		extras.MailGatewayToken = strings.TrimSpace(mc.GatewayToken)
	}
	return extras
}

func newDiscoveryToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "obd_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Server) handleDiscoverProject(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]string{"error": "bearer_required"})
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p := obelisk.FindProject(spec, id)
	if p == nil || !p.DiscoveryEnabled {
		writeJSONStatus(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	stored := disc.ReadToken(cfg, id)
	if !disc.ValidToken(stored, token) {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		return
	}
	rec := s.reconciler(cfg)
	ns := instanceNamespace()
	states := s.cachedServiceStates(r.Context(), spec, rec)
	resolved := obelisk.SpecWithResolvedCredentials(spec, func(inst obelisk.ServiceInstance) obelisk.InstanceCredentials {
		return credentials.MergeInstanceCredentials(r.Context(), ns, inst)
	})
	observedAt := time.Now().UTC()
	if !s.statesCache.at.IsZero() {
		observedAt = s.statesCache.at.UTC()
	}
	extras := projectDiscoveryExtras(cfg)
	audience := resolveDiscoveryAudience(r)
	out := obelisk.BuildProjectDiscovery(resolved, *p, states, observedAt, extras, audience)
	s.recordAudit("Project", "discovery_access", p.Name)
	writeJSON(w, out)
}

// resolveDiscoveryAudience picks edge vs in-cluster URLs. An explicit ?audience=edge|in_cluster
// always wins; otherwise ("" or "auto") it is inferred from how the caller reached us, so clients
// need no audience parameter: requests addressed to a cluster-internal service DNS name
// (*.svc, *.svc.cluster.local) get in-cluster URLs, everything else gets edge URLs.
func resolveDiscoveryAudience(r *http.Request) string {
	a := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("audience")))
	if a != "" && a != "auto" {
		return a
	}
	host := strings.ToLower(r.Host)
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	if strings.HasSuffix(host, ".svc") || strings.Contains(host, ".svc.") {
		return obelisk.ConnectionAudienceInCluster
	}
	return obelisk.ConnectionAudienceEdge
}
