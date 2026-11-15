package blueprint

import (
	"context"
	"fmt"
	"path/filepath"
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
