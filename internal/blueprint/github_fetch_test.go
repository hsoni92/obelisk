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
	tree := map[string]string{
		"commit-a": "tree-a",
		"commit-b": "tree-b",
	}
	bundles := map[string]map[string][]byte{
		"tree-a": {
			FileManifest: []byte(`{"blueprint_schema_version":1,"generation":1}`),
			FilePublic:   []byte("apiVersion: v1\n"),
			FileSecrets:  []byte("sealed-a"),
		},
		"tree-b": {
			FileManifest: []byte(`{"blueprint_schema_version":1,"generation":2}`),
			FilePublic:   []byte("apiVersion: v1\n"),
			FileSecrets:  []byte("sealed-b"),
		},
	}
	blobID := map[string]map[string]string{
		"tree-a": {FileManifest: "ma", FilePublic: "pa", FileSecrets: "sa"},
		"tree-b": {FileManifest: "mb", FilePublic: "pb", FileSecrets: "sb"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": head}})
			head = "commit-b"
		case strings.Contains(r.URL.Path, "/git/commits/"):
			sha := strings.TrimPrefix(r.URL.Path, "/git/commits/")
			if i := strings.Index(sha, "/"); i >= 0 {
				sha = sha[:i]
			}
			sha = strings.TrimPrefix(sha, "")
			for commit, treeSHA := range tree {
				if strings.HasSuffix(r.URL.Path, commit) {
					_ = json.NewEncoder(w).Encode(map[string]any{"tree": map[string]string{"sha": treeSHA}})
					return
				}
			}
			http.NotFound(w, r)
		case strings.Contains(r.URL.Path, "/git/trees/"):
			for treeSHA, files := range bundles {
				if strings.HasSuffix(r.URL.Path, treeSHA) {
					var ents []map[string]string
					for path := range files {
						ents = append(ents, map[string]string{
							"path": path, "type": "blob", "sha": blobID[treeSHA][path],
						})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"tree": ents})
					return
				}
			}
			http.NotFound(w, r)
		case strings.Contains(r.URL.Path, "/git/blobs/"):
			for treeSHA, ids := range blobID {
				for path, id := range ids {
					if strings.HasSuffix(r.URL.Path, id) {
						_ = json.NewEncoder(w).Encode(map[string]string{
							"encoding": "base64",
							"content":  base64.StdEncoding.EncodeToString(bundles[treeSHA][path]),
						})
						return
					}
				}
			}
			http.NotFound(w, r)
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
}