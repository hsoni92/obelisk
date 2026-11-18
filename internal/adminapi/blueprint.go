package adminapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
)

func (s *Server) handleBlueprintStatus(w http.ResponseWriter, r *http.Request) {
	if s.Blueprint == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": "blueprint runtime is not configured"})
		return
	}
	login, _ := s.Blueprint.Authenticate(r)
	writeJSON(w, s.Blueprint.Status(r.Context(), login))
}

func (s *Server) handleBlueprintGitHubConfig(w http.ResponseWriter, _ *http.Request) {
	if s.Blueprint == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": "blueprint runtime is not configured"})
		return
	}
	writeJSON(w, map[string]any{
		"client_configured": s.Blueprint.OAuth.Config.Configured(),
		"scope":             "repo",
	})
}

func (s *Server) handleBlueprintAuthorize(w http.ResponseWriter, r *http.Request) {
	if s.Blueprint == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": "blueprint runtime is not configured"})
		return
	}
	var body struct {
		Flow        string `json:"flow"`
		RedirectURI string `json:"redirect_uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.Flow == "" {
		body.Flow = "code"
	}
	var start blueprint.AuthorizeStart
	var err error
	switch body.Flow {
	case "device":
		start, err = s.Blueprint.OAuth.StartDevice(r.Context())
	default:
		start, err = s.Blueprint.OAuth.StartCode(body.RedirectURI)
	}
	if err != nil {
		writeBlueprintErr(w, err)
		return
	}
	writeJSON(w, start)
}

func (s *Server) handleBlueprintCallback(w http.ResponseWriter, r *http.Request) {
	if s.Blueprint == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": "blueprint runtime is not configured"})
		return
	}
	var body struct {
		Code       string `json:"code"`
		State      string `json:"state"`
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var tok blueprint.Token
	var err error
	if body.DeviceCode != "" {
		tok, err = s.Blueprint.OAuth.PollDevice(r.Context(), body.DeviceCode)
	} else {
		tok, err = s.Blueprint.OAuth.ExchangeCode(r.Context(), body.Code, body.State)
	}
	if errors.Is(err, blueprint.ErrPending) {
		writeJSONStatus(w, http.StatusAccepted, map[string]string{"status": "pending"})
		return
	}
	if err != nil {
		writeBlueprintErr(w, err)
		return
	}
	sid, err := s.Blueprint.CompleteLogin(r.Context(), tok)
	if err != nil {
		writeBlueprintErr(w, err)
		return
	}
	setSessionCookie(w, r, sid)
	saved, _ := s.Blueprint.Tokens.Load()
	writeJSON(w, map[string]any{"status": "ok", "login": saved.Login})
}

func (s *Server) handleBlueprintLogout(w http.ResponseWriter, r *http.Request) {
	if s.Blueprint != nil {
		s.Blueprint.Logout(blueprint.SessionCookie(r))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     blueprint.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, map[string]string{"status": "signed_out"})
}

func (s *Server) handleBlueprintOnboard(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Repository string `json:"repository"`
		Branch     string `json:"branch"`
		Passphrase string `json:"passphrase"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.Blueprint.Onboard(r.Context(), body.Repository, body.Branch, body.Passphrase); err != nil {
		writeBlueprintErr(w, err)
		return
	}
	writeJSON(w, s.Blueprint.Status(r.Context(), loginFrom(r)))
}

func (s *Server) handleBlueprintUnlock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Passphrase string `json:"passphrase"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.Blueprint.Unlock(body.Passphrase); err != nil {
		writeBlueprintErr(w, err)
		return
	}
	writeJSON(w, s.Blueprint.Status(r.Context(), loginFrom(r)))
}

func (s *Server) handleBlueprintLock(w http.ResponseWriter, r *http.Request) {
	if err := s.Blueprint.Lock(); err != nil {
		writeBlueprintErr(w, err)
		return
	}
	writeJSON(w, s.Blueprint.Status(r.Context(), loginFrom(r)))
}

func (s *Server) handleBlueprintImport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Passphrase      string `json:"passphrase"`
		Repository      string `json:"repository"`
		MachineIDChoice string `json:"machine_id_choice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.Blueprint.Import(r.Context(), body.Passphrase, body.Repository, body.MachineIDChoice); err != nil {
		writeBlueprintErr(w, err)
		return
	}
	s.Blueprint.InvalidateProbe()
	controller.ReconcileAsync(s.reconciler(config.EnsureReadablePath(s.ConfigPath)))
	writeJSON(w, s.Blueprint.Status(r.Context(), loginFrom(r)))
}

func (s *Server) handleBlueprintPush(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Force bool `json:"force"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	var err error
	if body.Force {
		err = s.Blueprint.Sync.ForcePush(r.Context())
	} else {
		err = s.Blueprint.Sync.PushNow(r.Context())
	}
	s.Blueprint.InvalidateProbe()
	if err != nil {
		writeBlueprintErr(w, err)
		return
	}
	writeJSON(w, s.Blueprint.Status(r.Context(), loginFrom(r)))
}

func loginFrom(r *http.Request) string {
	return loginFromContext(r)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, id string) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     blueprint.CookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}

func writeBlueprintErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, blueprint.ErrClientNotConfigured):
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	case errors.Is(err, blueprint.ErrWrongPassphrase), errors.Is(err, blueprint.ErrMachineChoice):
		writeJSONStatus(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, blueprint.ErrLocked), errors.Is(err, blueprint.ErrOnboardingRequired), errors.Is(err, blueprint.ErrImportRequired):
		writeJSONStatus(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, blueprint.ErrBundleCorrupt), errors.Is(err, blueprint.ErrTamper):
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
