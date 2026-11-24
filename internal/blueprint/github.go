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
	// Response metadata used to explain the failure. Zero values mean GitHub
	// did not send the header.
	ResetAt    time.Time     // X-RateLimit-Reset
	RetryAfter time.Duration // Retry-After
	Remaining  int           // X-RateLimit-Remaining; -1 when absent
	SSO        string        // X-GitHub-SSO (SAML SSO authorization required)
	RequestID  string        // X-GitHub-Request-Id, useful for GitHub Support
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

// Fetch reads the three bundle files from a single commit. ref may be a branch
// name or a 40-character commit SHA. Resolving a branch uses one git ref lookup,
// then all contents reads use that commit so concurrent pushes cannot mix generations.
func (g *GitHub) Fetch(ctx context.Context, repository, ref string) (map[string][]byte, error) {
	if ref == "" {
		ref = "main"
	}
	commitSHA, err := g.resolveCommitSHA(ctx, repository, ref)
	if err != nil {
		var se *StatusError
		if errors.As(err, &se) && se.Code == http.StatusNotFound {
			return nil, ErrEmptyRepo
		}
		return nil, err
	}
	return g.fetchBundleAtCommit(ctx, repository, commitSHA)
}

// fetchBundleAtCommit loads bundle blobs from the git tree (not the Contents API).
// Contents can differ from git objects for the same commit; import must match git push.
func (g *GitHub) fetchBundleAtCommit(ctx context.Context, repository, commitSHA string) (map[string][]byte, error) {
	owner, name := splitRepo(repository)
	var commit struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	uCommit := fmt.Sprintf("%s/repos/%s/%s/git/commits/%s", g.api(), url.PathEscape(owner), url.PathEscape(name), commitSHA)
	if err := g.do(ctx, http.MethodGet, uCommit, nil, &commit); err != nil {
		return nil, err
	}
	if commit.Tree.SHA == "" {
		return nil, fmt.Errorf("github commit %s has no tree", commitSHA[:min(8, len(commitSHA))])
	}
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}
	uTree := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s", g.api(), url.PathEscape(owner), url.PathEscape(name), commit.Tree.SHA)
	if err := g.do(ctx, http.MethodGet, uTree, nil, &tree); err != nil {
		return nil, err
	}
	blobs := map[string]string{}
	for _, ent := range tree.Tree {
		if ent.Type != "blob" {
			continue
		}
		blobs[ent.Path] = ent.SHA
	}
	required := []string{FileManifest, FilePublic, FileSecrets}
	out := make(map[string][]byte, len(required))
	for _, path := range required {
		blobSHA, ok := blobs[path]
		if !ok || blobSHA == "" {
			if path == FileManifest {
				return nil, ErrEmptyRepo
			}
			return nil, ErrEmptyRepo
		}
		b, err := g.getGitBlob(ctx, owner, name, blobSHA)
		if err != nil {
			return nil, err
		}
		out[path] = b
	}
	return out, nil
}

func (g *GitHub) getGitBlob(ctx context.Context, owner, repo, blobSHA string) ([]byte, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/git/blobs/%s", g.api(), url.PathEscape(owner), url.PathEscape(repo), blobSHA)
	var blob struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := g.do(ctx, http.MethodGet, u, nil, &blob); err != nil {
		return nil, err
	}
	if blob.Encoding != "base64" {
		return nil, fmt.Errorf("github blob encoding %q", blob.Encoding)
	}
	cleaned := strings.ReplaceAll(blob.Content, "\n", "")
	return base64.StdEncoding.DecodeString(cleaned)
}

func (g *GitHub) resolveCommitSHA(ctx context.Context, repository, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if isCommitSHA(ref) {
		return ref, nil
	}
	owner, name := splitRepo(repository)
	u := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", g.api(), url.PathEscape(owner), url.PathEscape(name), url.PathEscape(ref))
	var out struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := g.do(ctx, http.MethodGet, u, nil, &out); err != nil {
		return "", err
	}
	if out.Object.SHA == "" {
		return "", fmt.Errorf("github ref %q has no commit", ref)
	}
	return out.Object.SHA, nil
}

func isCommitSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
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
	emptyBranch := false
	// replacingTip is true when the new commit does not descend from the current
	// branch tip (same-UTC-day replace keeps the tip's parent). Such a ref update
	// is only accepted by GitHub with force=true.
	replacingTip := false
	if err := g.do(ctx, http.MethodGet, refGetURL, nil, &ref); err != nil {
		var se *StatusError
		if errors.As(err, &se) && se.Code == http.StatusNotFound {
			emptyBranch = true
		} else {
			return "", err
		}
	} else {
		tip := ref.Object.SHA
		manifest := files[FileManifest]
		if manifest == nil {
			manifest = []byte{}
		}
		p, err := g.resolveRetentionParent(ctx, repository, tip, manifest)
		if err != nil {
			return "", err
		}
		parent = p
		replacingTip = parent != tip
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
		patch := map[string]any{"sha": created.SHA, "force": replacingTip}
		if err := g.do(ctx, http.MethodPatch, refUpdateURL, patch, nil); err != nil {
			return "", err
		}
	}
	return created.SHA, nil
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
		return newStatusError(resp, string(b))
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
