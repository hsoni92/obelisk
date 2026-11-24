package blueprint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestGitHubPush_existingBranch(t *testing.T) {
	parentSHA := "parent-sha"
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": parentSHA}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/commits/"+parentSHA):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tree":      map[string]string{"sha": "base-tree"},
				"parents":   []any{},
				"committer": map[string]string{"date": "2020-01-01T00:00:00Z"},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/trees/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"tree": []any{}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/blobs"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "blob-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/trees"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "tree-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/commits"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "commit-sha"})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/git/refs/heads/main"):
			// GitHub rejects non-fast-forward ref updates with 422 unless force is set.
			// The new commit's parent is the current tip here, so a plain update
			// is a fast-forward and needs no force flag.
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	g := &GitHub{
		APIBase: srv.URL,
		Client:  srv.Client(),
		Token:   func(context.Context) (string, error) { return "tok", nil },
	}
	sha, err := g.Push(context.Background(), "acme/appliance", "main", "gen 1", map[string][]byte{
		FileManifest: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if sha != "commit-sha" {
		t.Fatalf("sha=%q", sha)
	}
}

// TestGitHubPush_sameDayReplaceForcesRef covers the same-UTC-day path: the new
// commit's parent is the tip's parent, not the tip, so the ref update must be
// forced or GitHub returns 422 "Update is not a fast forward".
func TestGitHubPush_sameDayReplaceForcesRef(t *testing.T) {
	const tip = "tip-sha"
	const tipParent = "tip-parent-sha"
	var mu sync.Mutex
	var patchForce *bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": tip}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/commits/"+tip):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tree":      map[string]string{"sha": "tip-tree"},
				"parents":   []map[string]string{{"sha": tipParent}},
				"committer": map[string]string{"date": "2026-03-10T12:00:00Z"},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/commits/"+tipParent):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tree":      map[string]string{"sha": "parent-tree"},
				"parents":   []any{},
				"committer": map[string]string{"date": "2026-03-09T12:00:00Z"},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/trees/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"tree": []any{}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/blobs"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "blob-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/trees"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "tree-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/commits"):
			var body struct {
				Parents []string `json:"parents"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Parents) != 1 || body.Parents[0] != tipParent {
				t.Errorf("parents=%v, want [%s] (same-day replace)", body.Parents, tipParent)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "commit-sha"})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/git/refs/heads/main"):
			var body struct {
				Force bool `json:"force"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			patchForce = &body.Force
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	g := &GitHub{
		APIBase: srv.URL,
		Client:  srv.Client(),
		Token:   func(context.Context) (string, error) { return "tok", nil },
	}
	manifest := []byte(`{"generation":2,"committed_at":"2026-03-10T15:00:00Z"}`)
	if _, err := g.Push(context.Background(), "acme/appliance", "main", "gen 2", map[string][]byte{
		FileManifest: manifest,
	}); err != nil {
		t.Fatal(err)
	}
	if patchForce == nil || !*patchForce {
		t.Fatalf("ref update force=%v, want true when replacing the tip", patchForce)
	}
}

func TestGitHubPush_emptyBranch(t *testing.T) {
	var mu sync.Mutex
	refExists := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/heads/main"):
			if !refExists {
				http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": "parent-sha"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/blobs"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "blob-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/trees"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "tree-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/commits"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "commit-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs"):
			refExists = true
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	g := &GitHub{
		APIBase: srv.URL,
		Client:  srv.Client(),
		Token:   func(context.Context) (string, error) { return "tok", nil },
	}
	sha, err := g.Push(context.Background(), "acme/appliance", "main", "gen 1", map[string][]byte{
		FileManifest: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if sha != "commit-sha" {
		t.Fatalf("sha=%q", sha)
	}
	if !refExists {
		t.Fatal("expected refs/heads/main to be created")
	}
}
