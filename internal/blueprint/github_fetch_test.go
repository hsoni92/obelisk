package blueprint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestGitHubFetch_pinsCommitSHA(t *testing.T) {
	var mu sync.Mutex
	head := "commit-a"
	files := map[string]map[string][]byte{
		"commit-a": {
			FileManifest: []byte(`{"blueprint_schema_version":1,"generation":1}`),
			FilePublic:   []byte("apiVersion: v1\n"),
			FileSecrets:  []byte("sealed-a"),
		},
		"commit-b": {
			FileManifest: []byte(`{"blueprint_schema_version":1,"generation":2}`),
			FilePublic:   []byte("apiVersion: v1\n"),
			FileSecrets:  []byte("sealed-b"),
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": head}})
			// Concurrent push on GitHub after ref resolve must not affect this fetch.
			head = "commit-b"
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/"):
			ref := r.URL.Query().Get("ref")
			path := strings.TrimPrefix(r.URL.Path, "/")
			path = path[strings.Index(path, "/contents/")+len("/contents/"):]
			if i := strings.Index(path, "?"); i >= 0 {
				path = path[:i]
			}
			set, ok := files[ref]
			if !ok {
				http.NotFound(w, r)
				return
			}
			body, ok := set[path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"encoding": "base64",
				"content":  base64.StdEncoding.EncodeToString(body),
			})
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

	out, err := g.Fetch(context.Background(), "acme/obelisk-appliance", "main")
	if err != nil {
		t.Fatal(err)
	}
	if string(out[FileSecrets]) != "sealed-a" {
		t.Fatalf("secrets = %q, want sealed-a (pinned commit-a)", out[FileSecrets])
	}
	if !strings.Contains(string(out[FileManifest]), `"generation":1`) {
		t.Fatalf("manifest = %s", out[FileManifest])
	}
}