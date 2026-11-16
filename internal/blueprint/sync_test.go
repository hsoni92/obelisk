package blueprint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

type memRemote struct {
	mu        sync.Mutex
	files     map[string][]byte
	pushes    int
	failUntil int
}

func (m *memRemote) EnsureRepo(context.Context, string) (string, error) {
	return "octo/obelisk-appliance", nil
}

func (m *memRemote) Fetch(context.Context, string, string) (map[string][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.files == nil {
		return nil, ErrEmptyRepo
	}
	out := map[string][]byte{}
	for k, v := range m.files {
		out[k] = append([]byte(nil), v...)
	}
	return out, nil
}

func (m *memRemote) Push(_ context.Context, _, _, _ string, files map[string][]byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pushes++
	if m.pushes <= m.failUntil {
		return "", fmt.Errorf("temporary github outage")
	}
	m.files = map[string][]byte{}
	for k, v := range files {
		m.files[k] = append([]byte(nil), v...)
	}
	return fmt.Sprintf("sha%d", m.pushes), nil
}

func (m *memRemote) pushCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushes
}

func newTestSyncer(t *testing.T, remote Remote) (*Syncer, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-sync001"
	spec.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
	spec.Blueprint.GitHub.Branch = "main"
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}
	kdf := testKDF(t)
	key, err := DeriveKey("phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	session := &Session{}
	session.Set(key, kdf)
	s := NewSyncer(path, session, remote)
	s.Debounce = 30 * time.Millisecond
	s.Backoff = 20 * time.Millisecond
	return s, path
}

func TestSyncCoalescesSaves(t *testing.T) {
	remote := &memRemote{}
	s, _ := newTestSyncer(t, remote)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Loop(ctx)
	for i := 0; i < 5; i++ {
		s.Enqueue()
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && remote.pushCount() < 1 {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
	if got := remote.pushCount(); got != 1 {
		t.Fatalf("pushes = %d, want 1", got)
	}
}

func TestSyncRetriesFailures(t *testing.T) {
	remote := &memRemote{failUntil: 2}
	s, _ := newTestSyncer(t, remote)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Loop(ctx)
	s.Enqueue()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && remote.pushCount() < 3 {
		time.Sleep(20 * time.Millisecond)
	}
	if got := remote.pushCount(); got < 3 {
		t.Fatalf("pushes = %d, want at least 3", got)
	}
	if remote.files == nil {
		t.Fatal("expected a successful push to store files")
	}
}

func TestSyncRefusesRemoteGenerationAhead(t *testing.T) {
	remote := &memRemote{files: map[string][]byte{
		FileManifest: []byte(`{"blueprint_schema_version":1,"generation":9,"content_sha256":"different","machine_id":"obelisk-sync001"}`),
		FilePublic:   []byte("apiVersion: obelisk.dev/v1\n"),
		FileSecrets:  []byte("cipher"),
	}}
	s, path := newTestSyncer(t, remote)
	spec, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	spec.Blueprint.GitHub.Generation = 5
	spec.Blueprint.GitHub.LastCommitSHA = "abc"
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}
	err = s.PushNow(context.Background())
	if err != ErrImportRequired {
		t.Fatalf("err = %v", err)
	}
	if remote.pushCount() != 0 {
		t.Fatalf("pushed over a newer remote generation")
	}
}

func TestFailSkipsImportPersistWhenNeverSynced(t *testing.T) {
	s, path := newTestSyncer(t, &memRemote{})
	err := s.fail(ErrImportRequired)
	if err != ErrImportRequired {
		t.Fatalf("err = %v", err)
	}
	spec, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Blueprint.GitHub.LastPushError != "" {
		t.Fatalf("last_push_error = %q", spec.Blueprint.GitHub.LastPushError)
	}
}

func TestSyncOverwritesJunkRemoteWhenNeverSynced(t *testing.T) {
	remote := &memRemote{files: map[string][]byte{
		FileManifest: []byte(`{"blueprint_schema_version":1,"generation":9,"content_sha256":"different","machine_id":"obelisk-sync001"}`),
		FilePublic:   []byte("apiVersion: obelisk.dev/v1\n"),
		FileSecrets:  []byte("cipher"),
	}}
	s, path := newTestSyncer(t, remote)
	err := s.PushNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spec, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Blueprint.GitHub.Generation != 10 {
		t.Fatalf("generation = %d", spec.Blueprint.GitHub.Generation)
	}
}

func TestSyncForcePushOverwritesRemoteAhead(t *testing.T) {
	remote := &memRemote{files: map[string][]byte{
		FileManifest: []byte(`{"blueprint_schema_version":1,"generation":105,"content_sha256":"different","machine_id":"obelisk-sync001"}`),
		FilePublic:   []byte("apiVersion: obelisk.dev/v1\n"),
		FileSecrets:  []byte("cipher"),
	}}
	s, path := newTestSyncer(t, remote)
	spec, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	spec.Blueprint.GitHub.Generation = 103
	spec.Blueprint.GitHub.LastCommitSHA = "abc"
	spec.Blueprint.GitHub.SyncStatus = obelisk.SyncFailed
	spec.Blueprint.GitHub.LastPushError = ErrImportRequired.Error()
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}
	if err := s.ForcePush(context.Background()); err != nil {
		t.Fatalf("ForcePush err = %v", err)
	}
	if remote.pushCount() != 1 {
		t.Fatalf("pushes = %d, want 1", remote.pushCount())
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Blueprint.GitHub.Generation != 106 {
		t.Fatalf("generation = %d, want 106", got.Blueprint.GitHub.Generation)
	}
	if got.Blueprint.GitHub.SyncStatus != obelisk.SyncSynced {
		t.Fatalf("sync_status = %q, want %q", got.Blueprint.GitHub.SyncStatus, obelisk.SyncSynced)
	}
	if got.Blueprint.GitHub.LastPushError != "" {
		t.Fatalf("last_push_error = %q, want empty", got.Blueprint.GitHub.LastPushError)
	}
}

func TestImportUsesUnlockedSessionKeyWithoutPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	local := obelisk.DefaultSpec("vps")
	local.Machine.ID = "obelisk-same01"
	local.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
	local.Blueprint.GitHub.Branch = "main"
	local.Blueprint.GitHub.Generation = 103
	local.Blueprint.GitHub.LastCommitSHA = "sha103"
	if err := config.SaveQuiet(path, local); err != nil {
		t.Fatal(err)
	}

	kdf := testKDF(t)
	key, err := DeriveKey("phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	remoteSpec := local
	remoteSpec.Exposure.Funnel.RustFSS3.Enabled = true
	bundle, err := Pack(remoteSpec, Secrets{}, key, kdf, 105)
	if err != nil {
		t.Fatal(err)
	}

	srv := fakeGitHubContentsServer(t, bundle.Files)
	defer srv.Close()

	rt, err := NewRuntime(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	rt.Git = &GitHub{
		APIBase: srv.URL,
		Client:  srv.Client(),
		Token:   func(context.Context) (string, error) { return "tok", nil },
	}
	rt.Session.Set(key, kdf)

	if err := rt.Import(context.Background(), "", "", ""); err != nil {
		t.Fatalf("Import err = %v", err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Blueprint.GitHub.Generation != 105 {
		t.Fatalf("generation = %d, want 105", got.Blueprint.GitHub.Generation)
	}
	if !got.Exposure.Funnel.RustFSS3.Enabled {
		t.Fatal("expected imported spec change to be applied")
	}
}

func TestImportRejectsEmptyPassphraseWhenKeyDiffers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	local := obelisk.DefaultSpec("vps")
	local.Machine.ID = "obelisk-same01"
	local.Blueprint.GitHub.Repository = "octo/obelisk-appliance"
	local.Blueprint.GitHub.Branch = "main"
	if err := config.SaveQuiet(path, local); err != nil {
		t.Fatal(err)
	}

	kdf := testKDF(t)
	remoteKey, err := DeriveKey("remote-phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	localKey, err := DeriveKey("local-phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Pack(local, Secrets{}, remoteKey, kdf, 105)
	if err != nil {
		t.Fatal(err)
	}

	srv := fakeGitHubContentsServer(t, bundle.Files)
	defer srv.Close()

	rt, err := NewRuntime(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	rt.Git = &GitHub{
		APIBase: srv.URL,
		Client:  srv.Client(),
		Token:   func(context.Context) (string, error) { return "tok", nil },
	}
	rt.Session.Set(localKey, kdf)

	err = rt.Import(context.Background(), "", "", "")
	if !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("err = %v, want ErrWrongPassphrase", err)
	}
}

func fakeGitHubContentsServer(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, content := range files {
			if strings.HasSuffix(r.URL.Path, "/contents/"+name) {
				_ = json.NewEncoder(w).Encode(map[string]string{
					"encoding": "base64",
					"content":  base64.StdEncoding.EncodeToString(content),
				})
				return
			}
		}
		http.NotFound(w, r)
	}))
}


