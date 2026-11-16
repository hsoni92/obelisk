package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type stubRemote struct {
	mu     sync.Mutex
	files  map[string][]byte
	pushes int
}

func (s *stubRemote) EnsureRepo(context.Context, string) (string, error) {
	return "octo/obelisk-appliance", nil
}

func (s *stubRemote) Fetch(context.Context, string, string) (map[string][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		return nil, blueprint.ErrEmptyRepo
	}
	out := map[string][]byte{}
	for k, v := range s.files {
		out[k] = append([]byte(nil), v...)
	}
	return out, nil
}

func (s *stubRemote) Push(_ context.Context, _, _, _ string, files map[string][]byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pushes++
	s.files = map[string][]byte{}
	for k, v := range files {
		s.files[k] = append([]byte(nil), v...)
	}
	return "new-sha", nil
}

func TestHandleBlueprintPush_Force(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-api01"
	spec.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
	spec.Blueprint.GitHub.Branch = "main"
	spec.Blueprint.GitHub.Generation = 103
	spec.Blueprint.GitHub.LastCommitSHA = "sha103"
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}

	rt, err := blueprint.NewRuntime(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	kdf := blueprint.KDFParams{
		Memory:      1024,
		Iterations:  1,
		Parallelism: 1,
		Salt:        []byte("0123456789abcdef"),
	}
	key, err := blueprint.DeriveKey("phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	rt.Session.Set(key, kdf)
	remote := &stubRemote{files: map[string][]byte{
		blueprint.FileManifest: []byte(`{"blueprint_schema_version":1,"generation":105,"content_sha256":"different","machine_id":"obelisk-api01"}`),
		blueprint.FilePublic:   []byte("apiVersion: obelisk.dev/v1\n"),
		blueprint.FileSecrets:  []byte("cipher"),
	}}
	rt.Sync = blueprint.NewSyncer(path, rt.Session, remote)

	srv := &Server{ConfigPath: path, HelmRoot: dir, Blueprint: rt}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/blueprint/push", nil)
	rr := httptest.NewRecorder()
	srv.handleBlueprintPush(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("normal push status = %d, want 409 (%s)", rr.Code, rr.Body.String())
	}

	reqForce := httptest.NewRequest(http.MethodPost, "/api/v1/blueprint/push", bytes.NewBufferString(`{"force":true}`))
	rrForce := httptest.NewRecorder()
	srv.handleBlueprintPush(rrForce, reqForce)
	if rrForce.Code != http.StatusOK {
		t.Fatalf("force push status = %d, want 200 (%s)", rrForce.Code, rrForce.Body.String())
	}
	var st blueprint.Status
	if err := json.Unmarshal(rrForce.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st.Generation != 106 {
		t.Fatalf("generation = %d, want 106", st.Generation)
	}
	if st.ImportRequired {
		t.Fatal("expected import_required = false after force push")
	}
	if st.SyncStatus != obelisk.SyncSynced {
		t.Fatalf("sync_status = %q, want synced", st.SyncStatus)
	}
}
