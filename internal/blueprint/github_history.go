package blueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// blueprintMaxBranchCommits caps linear history per branch. With same-day
	// coalescing, one commit per UTC day yields a rolling ~30-day window.
	blueprintMaxBranchCommits  = 30
	blueprintAncestorWalkLimit = blueprintMaxBranchCommits + 5
)

func utcDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func manifestUTCDay(manifestJSON []byte) (time.Time, bool) {
	var m Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil || strings.TrimSpace(m.CommittedAt) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, m.CommittedAt)
	if err != nil {
		return time.Time{}, false
	}
	return utcDay(t), true
}

func parseGitHubTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return utcDay(t), true
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return utcDay(t), true
	}
	return time.Time{}, false
}

// selectRetentionParent picks the git parent for a new blueprint commit.
// ancestors is a linear history newest-first: [tip, tip.parent, ...].
// Depth is capped at blueprintMaxBranchCommits (30). With at most one commit
// per UTC day (same-day replace), that matches a rolling 30-day window.
func selectRetentionParent(ancestors []string, tipDay, newDay time.Time) string {
	if len(ancestors) == 0 {
		return ""
	}
	var parent string
	if tipDay.Equal(newDay) {
		if len(ancestors) >= 2 {
			parent = ancestors[1]
		}
	} else {
		parent = ancestors[0]
	}
	for parent != "" {
		idx := indexOfSHA(ancestors, parent)
		if idx < 0 {
			break
		}
		depth := len(ancestors) - idx
		if depth >= blueprintMaxBranchCommits {
			if idx+1 < len(ancestors) {
				parent = ancestors[idx+1]
				continue
			}
			parent = ""
			continue
		}
		break
	}
	return parent
}

func indexOfSHA(list []string, sha string) int {
	for i, s := range list {
		if s == sha {
			return i
		}
	}
	return -1
}

type gitCommitMeta struct {
	Tree    string
	Parents []string
	Date    time.Time
}

func (g *GitHub) getCommitMeta(ctx context.Context, owner, repo, sha string) (gitCommitMeta, error) {
	var raw struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
		Parents []struct {
			SHA string `json:"sha"`
		} `json:"parents"`
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	}
	u := fmt.Sprintf("%s/repos/%s/%s/git/commits/%s", g.api(), owner, repo, sha)
	if err := g.do(ctx, http.MethodGet, u, nil, &raw); err != nil {
		return gitCommitMeta{}, err
	}
	meta := gitCommitMeta{Tree: raw.Tree.SHA}
	for _, p := range raw.Parents {
		if p.SHA != "" {
			meta.Parents = append(meta.Parents, p.SHA)
		}
	}
	if d, ok := parseGitHubTime(raw.Committer.Date); ok {
		meta.Date = d
	}
	return meta, nil
}

func (g *GitHub) linearAncestors(ctx context.Context, repository, tip string) ([]string, error) {
	owner, name := splitRepo(repository)
	var out []string
	cur := tip
	for len(out) < blueprintAncestorWalkLimit && cur != "" {
		out = append(out, cur)
		meta, err := g.getCommitMeta(ctx, owner, name, cur)
		if err != nil {
			return nil, err
		}
		if len(meta.Parents) == 0 {
			break
		}
		cur = meta.Parents[0]
	}
	return out, nil
}

func (g *GitHub) commitUTCday(ctx context.Context, repository, commitSHA string) (time.Time, error) {
	files, err := g.fetchBundleAtCommit(ctx, repository, commitSHA)
	if err != nil {
		return g.fallbackCommitterDay(ctx, repository, commitSHA)
	}
	if day, ok := manifestUTCDay(files[FileManifest]); ok {
		return day, nil
	}
	return g.fallbackCommitterDay(ctx, repository, commitSHA)
}

func (g *GitHub) fallbackCommitterDay(ctx context.Context, repository, commitSHA string) (time.Time, error) {
	owner, name := splitRepo(repository)
	meta, err := g.getCommitMeta(ctx, owner, name, commitSHA)
	if err != nil {
		return time.Time{}, err
	}
	if !meta.Date.IsZero() {
		return meta.Date, nil
	}
	return time.Time{}, fmt.Errorf("no date for commit %s", commitSHA[:min(8, len(commitSHA))])
}

func (g *GitHub) resolveRetentionParent(ctx context.Context, repository, tip string, newManifest []byte) (string, error) {
	ancestors, err := g.linearAncestors(ctx, repository, tip)
	if err != nil {
		return "", err
	}
	newDay, ok := manifestUTCDay(newManifest)
	if !ok {
		newDay = utcDay(time.Now().UTC())
	}
	tipDay, err := g.commitUTCday(ctx, repository, tip)
	if err != nil {
		return "", err
	}
	return selectRetentionParent(ancestors, tipDay, newDay), nil
}