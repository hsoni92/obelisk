package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/hsoni92/obelisk/internal/release"
)

type installMeta = release.InstallMeta

func readInstallMeta(root string) (installMeta, string, error) { return release.ReadInstallMeta(root) }

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
