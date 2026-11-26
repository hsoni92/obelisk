// Package release resolves Obelisk release builds and reads the installed build. It is shared
// by the obelisk CLI (`obelisk update --check`) and the controller's update-available alert.
package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultRepo = "hsoni92/obelisk"

var buildTagRE = regexp.MustCompile(`^build-([0-9]+)$`)

// ErrNoRelease means the repository has no published release yet.
var ErrNoRelease = errors.New("no release has been published yet")

// ParseBuildTag turns "build-142" into 142.
func ParseBuildTag(tag string) (int, bool) {
	m := buildTagRE.FindStringSubmatch(strings.TrimSpace(tag))
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

func BaseURL() string {
	if v := strings.TrimSpace(os.Getenv("OBELISK_RELEASE_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://github.com"
}

func Repo(metaRepo string) string {
	if v := strings.TrimSpace(os.Getenv("OBELISK_REPO")); v != "" {
		return v
	}
	if metaRepo != "" {
		return metaRepo
	}
	return DefaultRepo
}

// Resolve returns the build number of the latest release, or of the release tagged `tag` when
// tag is non-empty. It uses github.com web redirects, not the REST API, so there is no
// 60-requests-per-hour anonymous limit.
func Resolve(base, repo, tag string) (int, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if tag != "" {
		n, ok := ParseBuildTag(tag)
		if !ok {
			return 0, fmt.Errorf("invalid release %q (expected build-N)", tag)
		}
		resp, err := client.Get(fmt.Sprintf("%s/%s/releases/tag/%s", base, repo, tag))
		if err != nil {
			return 0, err
		}
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			return n, nil
		case http.StatusNotFound:
			return 0, fmt.Errorf("release %s not found in %s", tag, repo)
		default:
			return 0, fmt.Errorf("unexpected HTTP %d checking release %s", resp.StatusCode, tag)
		}
	}

	resp, err := client.Get(fmt.Sprintf("%s/%s/releases/latest", base, repo))
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusFound, http.StatusMovedPermanently, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		loc, err := url.Parse(resp.Header.Get("Location"))
		if err != nil {
			return 0, fmt.Errorf("bad redirect from releases/latest: %w", err)
		}
		n, ok := ParseBuildTag(path.Base(loc.Path))
		if !ok {
			return 0, fmt.Errorf("latest release %q is not a build-N tag", path.Base(loc.Path))
		}
		return n, nil
	case http.StatusNotFound:
		return 0, ErrNoRelease
	default:
		return 0, fmt.Errorf("unexpected HTTP %d from releases/latest", resp.StatusCode)
	}
}

// InstallMeta is <root>/.install-meta.json. Schema 2 adds build/tag/profile/arch/source.
// Files written before schema 2 have no build; they count as build 0.
type InstallMeta struct {
	Schema      int    `json:"schema,omitempty"`
	Repo        string `json:"repo"`
	Ref         string `json:"ref"`
	InstalledAt string `json:"installed_at"`
	ImageTag    string `json:"image_tag"`
	Build       int    `json:"build,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Arch        string `json:"arch,omitempty"`
	Source      string `json:"source,omitempty"` // "release" or "dev"
}

func ReadInstallMeta(root string) (InstallMeta, string, error) {
	metaPath := filepath.Join(root, ".install-meta.json")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		return InstallMeta{}, metaPath, err
	}
	var m InstallMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return InstallMeta{}, metaPath, err
	}
	return m, metaPath, nil
}
