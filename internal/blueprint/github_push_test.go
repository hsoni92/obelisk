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
			_ = json.NewEncoder(w).Encode(map[string]any{"tree": map[string]string{"sha": "base-tree"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/blobs"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "blob-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/trees"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "tree-sha"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/commits"):
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "commit-sha"})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/git/refs/heads/main"):
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