package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/credentials"
	"github.com/hsoni92/obelisk/internal/diagnostics"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type diagnosticsRunRequest struct {
	Instances []diagnostics.InstanceRef `json:"instances"`
}

func (s *Server) handleDiagnosticsRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req diagnosticsRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Instances) == 0 {
		http.Error(w, "instances required", http.StatusBadRequest)
		return
	}
	if len(req.Instances) > 32 {
		http.Error(w, "at most 32 instances per run", http.StatusBadRequest)
		return
	}

	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rec := s.reconciler(cfg)
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	states := rec.ObserveServiceStates(ctx, spec)
	ns := rec.Namespace
	if ns == "" {
		ns = instanceNamespace()
	}
	resolved := obelisk.SpecWithResolvedCredentials(spec, func(inst obelisk.ServiceInstance) obelisk.InstanceCredentials {
		return credentials.MergeInstanceCredentials(ctx, ns, inst)
	})
	conns := obelisk.BuildConnections(resolved, states)
	resp := diagnostics.Run(ctx, resolved, conns, states, req.Instances)
	writeJSON(w, resp)
}
