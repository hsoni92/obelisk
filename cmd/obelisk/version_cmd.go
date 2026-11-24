package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type installMeta struct {
	Repo        string `json:"repo"`
	Ref         string `json:"ref"`
	InstalledAt string `json:"installed_at"`
	ImageTag    string `json:"image_tag"`
}

func cmdVersion() {
	root := resolveObeliskRoot()
	metaPath := filepath.Join(root, ".install-meta.json")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		fmt.Printf("install-meta: (not found at %s)\n", metaPath)
		return
	}
	var m installMeta
	if err := json.Unmarshal(b, &m); err != nil {
		fmt.Printf("install-meta: %s (parse error: %v)\n", metaPath, err)
		return
	}
	fmt.Printf("repo=%s\nref=%s\ninstalled_at=%s\nimage_tag=%s\nroot=%s\n",
		m.Repo, m.Ref, m.InstalledAt, m.ImageTag, root)
}
