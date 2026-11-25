package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// installMeta is /opt/obelisk/.install-meta.json. Schema 2 adds build/tag/profile/arch/source.
// Files written before schema 2 have no build; they count as build 0.
type installMeta struct {
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

func readInstallMeta(root string) (installMeta, string, error) {
	metaPath := filepath.Join(root, ".install-meta.json")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		return installMeta{}, metaPath, err
	}
	var m installMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return installMeta{}, metaPath, err
	}
	return m, metaPath, nil
}

func cmdVersion() {
	root := resolveObeliskRoot()
	m, metaPath, err := readInstallMeta(root)
	embedded := buildNumber
	if embedded == "" {
		embedded = "dev"
	}
	fmt.Printf("cli_build=%s\ncli_commit=%s\ncli_arch=%s/%s\n", embedded, buildCommit, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("install-meta: (not found at %s)\n", metaPath)
		} else {
			fmt.Printf("install-meta: %s (parse error: %v)\n", metaPath, err)
		}
		return
	}
	fmt.Printf("build=%d\ntag=%s\nsource=%s\nprofile=%s\narch=%s\nrepo=%s\nref=%s\ninstalled_at=%s\nimage_tag=%s\nroot=%s\n",
		m.Build, m.Tag, m.Source, m.Profile, m.Arch, m.Repo, m.Ref, m.InstalledAt, m.ImageTag, root)
	if embedded != "dev" && m.Build != 0 && fmt.Sprint(m.Build) != embedded {
		fmt.Printf("warning: CLI build %s does not match installed build %d; run: sudo obelisk update --force\n", embedded, m.Build)
	}
}
