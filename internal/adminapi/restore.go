package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/internal/restore"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func (s *Server) handleRestoreChain(w http.ResponseWriter, r *http.Request) {
	if s.Blueprint == nil || s.Restore == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": "restore runtime is not configured"})
		return
	}
	var body struct {
		Passphrase      string `json:"passphrase"`
		Repository      string `json:"repository"`
		MachineIDChoice string `json:"machine_id_choice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ch := &blueprint.Chain{
		Repository: body.Repository,
		Choice:     body.MachineIDChoice,
		Passphrase: body.Passphrase,
	}
	state := s.Restore.Start(r.Context(), s.chainHooks(ch))
	if state.Error == "restore already running" {
		writeJSONStatus(w, http.StatusConflict, state)
		return
	}
	writeJSONStatus(w, http.StatusAccepted, state)
}

func (s *Server) handleRestoreChainStatus(w http.ResponseWriter, _ *http.Request) {
	if s.Restore == nil {
		writeJSON(w, map[string]string{"phase": "idle"})
		return
	}
	writeJSON(w, s.Restore.State())
}

func (s *Server) chainHooks(ch *blueprint.Chain) restore.Hooks {
	return restore.Hooks{
		Pull: func(ctx context.Context) error {
			return s.Blueprint.ChainPull(ctx, ch)
		},
		Unlock: func(context.Context) error {
			return s.Blueprint.ChainUnlock(ch)
		},
		Apply: func(context.Context) error {
			return s.Blueprint.ChainApply(ch)
		},
		RestoreData: func(ctx context.Context) (int, error) {
			if blueprint.ChainSkipData(ch) {
				return 0, nil
			}
			spec, err := config.Load(s.ConfigPath)
			if err != nil {
				return 0, err
			}
			return restore.RestoreDataFiles(ctx, s.ConfigPath, spec)
		},
		Reconcile: func(ctx context.Context) error {
			rec := controller.Reconciler{ConfigPath: s.ConfigPath, HelmRoot: s.HelmRoot}
			_, err := rec.Reconcile(ctx)
			return err
		},
		ForceFunnelOff: func(context.Context) error {
			return s.Blueprint.Sync.UpdateSpec(func(spec *obelisk.Spec) {
				spec.Exposure.Funnel.RustFSS3.Enabled = false
			})
		},
	}
}
