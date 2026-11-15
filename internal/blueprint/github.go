package blueprint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Remote is the git transport used by sync and restore.
type Remote interface {
	EnsureRepo(ctx context.Context, repository string) (fullName string, err error)
	Fetch(ctx context.Context, repository, branch string) (files map[string][]byte, err error)
	Push(ctx context.Context, repository, branch, message string, files map[string][]byte) (sha string, err error)
}

// StatusError is an HTTP error from the GitHub API.
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	msg := fmt.Sprintf("github http %d", e.Code)
	var body struct {
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(e.Body), &body) == nil && body.Message != "" {
		msg += ": " + body.Message
	}
	return msg
}

// GitHub implements Remote with the Git Data API (one commit per push) and the
// Contents API for reads. Requests use the OAuth access token over HTTPS.
type GitHub struct {
	Client  *http.Client
	APIBase string
	Token   func(ctx context.Context) (string, error)
}

func (g *GitHub) api() string {
	if g.APIBase != "" {
		return strings.TrimRight(g.APIBase, "/")
	}
	return "https://api.github.com"
}

func (g *GitHub) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// Login returns the authenticated GitHub user.
func (g *GitHub) Login(ctx context.Context) (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := g.do(ctx, http.MethodGet, g.api()+"/user", nil, &user); err != nil {
		return "", err
	}
	if user.Login == "" {
		return "", fmt.Errorf("github user has no login")
	}
	return user.Login, nil
}

// EnsureRepo returns owner/name, creating a private repo when the caller owns the name.
func (g *GitHub) EnsureRepo(ctx context.Context, repository string) (string, error) {
	full, login, err := g.resolveName(ctx, repository)
	if err != nil {
		return "", err
	}
	var existing struct {
		FullName string `json:"full_name"`
	}
	err = g.do(ctx, http.MethodGet, g.api()+"/repos/"+url.PathEscape(fullOwner(full))+"/"+url.PathEscape(fullName(full)), nil, &existing)
	if err == nil && existing.FullName != "" {
		return existing.FullName, nil
	}
	var se *StatusError
	if err != nil && (!errors.As(err, &se) || se.Code != http.StatusNotFound) {
		return "", err
	}
	owner := fullOwner(full)
	name := fullName(full)
	if owner != login {
		return "", fmt.Errorf("repository %s was not found", full)
	}
	var created struct {
		FullName string `json:"full_name"`
	}
	body := map[string]any{
		"name":        name,
		"private":     true,
		"auto_init":   true,
		"description": "Obelisk encrypted blueprint",
	}
	if err := g.do(ctx, http.MethodPost, g.api()+"/user/repos", body, &created); err != nil {
		return "", err
	}
	return created.FullName, nil
}

// LookupRepo checks whether owner/name exists and whether it contains a bundle.
// It does not create a repository.
func (g *GitHub) LookupRepo(ctx context.Context, repository, branch string) (present bool, manifest []byte, err error) {
	files, err := g.Fetch(ctx, repository, branch)
	if err != nil {
		if errors.Is(err, ErrEmptyRepo) {
			return false, nil, nil
		}
		var se *StatusError
		if errors.As(err, &se) && se.Code == http.StatusNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, files[FileManifest], nil
}

// Fetch reads the three bundle files. A missing manifest is ErrEmptyRepo.
func (g *GitHub) Fetch(ctx context.Context, repository, branch string) (map[string][]byte, error) {
	if branch == "" {
		branch = "main"
	}
	out := map[string][]byte{}
	for _, path := range []string{FileManifest, FilePublic, FileSecrets} {
		b, err := g.getFile(ctx, repository, branch, path)
		if err != nil {
			var se *StatusError
			if errors.As(err, &se) && se.Code == http.StatusNotFound {
				if path == FileManifest {
					return nil, ErrEmptyRepo
				}
				return nil, ErrEmptyRepo
			}
			return nil, err
		}
		out[path] = b
	}
	return out, nil
}

// Push writes all bundle files as a single commit on branch.
func (g *GitHub) Push(ctx context.Context, repository, branch, message string, files map[string][]byte) (string, error) {
	if branch == "" {
		branch = "main"
	}
	owner, name := splitRepo(repository)
	refGetURL := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", g.api(), url.PathEscape(owner), url.PathEscape(name), url.PathEscape(branch))
	refUpdateURL := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s", g.api(), url.PathEscape(owner), url.PathEscape(name), url.PathEscape(branch))
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	parent := ""
	baseTree := ""
	emptyBranch := false
	if err := g.do(ctx, http.MethodGet, refGetURL, nil, &ref); err != nil {
		var se *StatusError
		if errors.As(err, &se) && se.Code == http.StatusNotFound {
			emptyBranch = true
		} else {
			return "", err
		}
	} else {
		parent = ref.Object.SHA
		var commit struct {
			Tree struct {
				SHA string `json:"sha"`
			} `json:"tree"`
		}
		if err := g.do(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/%s/git/commits/%s", g.api(), url.PathEscape(owner), url.PathEscape(name), parent), nil, &commit); err != nil {
			return "", err
		}
		baseTree = commit.Tree.SHA
	}
	type treeEntry struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
	}
	var entries []treeEntry
	for path, content := range files {
		var blob struct {
			SHA string `json:"sha"`
		}
		body := map[string]string{
			"content":  base64.StdEncoding.EncodeToString(content),
			"encoding": "base64",
		}
		if err := g.do(ctx, http.MethodPost, fmt.Sprintf("%s/repos/%s/%s/git/blobs", g.api(), url.PathEscape(owner), url.PathEscape(name)), body, &blob); err != nil {
			return "", err
		}
		entries = append(entries, treeEntry{Path: path, Mode: "100644", Type: "blob", SHA: blob.SHA})
	}
	var tree struct {
		SHA string `json:"sha"`
	}
	treeBody := map[string]any{"tree": entries}
	if baseTree != "" {
		treeBody["base_tree"] = baseTree
	}
	if err := g.do(ctx, http.MethodPost, fmt.Sprintf("%s/repos/%s/%s/git/trees", g.api(), url.PathEscape(owner), url.PathEscape(name)), treeBody, &tree); err != nil {
		return "", err
	}
	var created struct {
		SHA string `json:"sha"`
	}
	commitBody := map[string]any{"message": message, "tree": tree.SHA}
	if parent != "" {
		commitBody["parents"] = []string{parent}
	}
	if err := g.do(ctx, http.MethodPost, fmt.Sprintf("%s/repos/%s/%s/git/commits", g.api(), url.PathEscape(owner), url.PathEscape(name)), commitBody, &created); err != nil {
		return "", err
	}
	if emptyBranch {
		refBody := map[string]string{"ref": "refs/heads/" + branch, "sha": created.SHA}
		if err := g.do(ctx, http.MethodPost, fmt.Sprintf("%s/repos/%s/%s/git/refs", g.api(), url.PathEscape(owner), url.PathEscape(name)), refBody, nil); err != nil {
			return "", err
		}
	} else {
		patch := map[string]string{"sha": created.SHA}
		if err := g.do(ctx, http.MethodPatch, refUpdateURL, patch, nil); err != nil {
			return "", err
		}
	}
	return created.SHA, nil
}

func (g *GitHub) getFile(ctx context.Context, repository, branch, path string) ([]byte, error) {
	owner, name := splitRepo(repository)
	u := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", g.api(), url.PathEscape(owner), url.PathEscape(name), path, url.QueryEscape(branch))
	var file struct {
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := g.do(ctx, http.MethodGet, u, nil, &file); err != nil {
		return nil, err
	}
	cleaned := strings.ReplaceAll(file.Content, "\n", "")
	return base64.StdEncoding.DecodeString(cleaned)
}

func (g *GitHub) resolveName(ctx context.Context, repository string) (full, login string, err error) {
	login, err = g.Login(ctx)
	if err != nil {
		return "", "", err
	}
	repository = strings.Trim(strings.TrimSpace(repository), "/")
	if repository == "" {
		repository = obeliskDefaultRepo
	}
	if !strings.Contains(repository, "/") {
		repository = login + "/" + repository
	}
	return repository, login, nil
}

const obeliskDefaultRepo = "obelisk-appliance"

func splitRepo(full string) (owner, name string) {
	owner, name, _ = strings.Cut(full, "/")
	return owner, name
}

func fullOwner(full string) string { o, _ := splitRepo(full); return o }
func fullName(full string) string  { _, n := splitRepo(full); return n }

func (g *GitHub) do(ctx context.Context, method, endpoint string, in any, out any) error {
	token, err := g.Token(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return &StatusError{Code: resp.StatusCode, Body: sanitize(string(b))}
	}
	if out != nil && len(bytes.TrimSpace(b)) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("github response: %w", err)
		}
	}
	return nil
}

func sanitize(s string) string {
	if len(s) > 300 {
		s = s[:300]
	}
	return tokenRE.ReplaceAllString(s, "[redacted]")
}
