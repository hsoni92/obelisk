package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	root := fs.String("root", resolveObeliskRoot(), "obelisk install tree (default /opt/obelisk or OBELISK_ROOT)")
	check := fs.Bool("check", false, "only report whether a newer build exists (exit 0 up to date, 10 update available, 11 cannot tell)")
	to := fs.String("to", "", "install this exact build (build-N). Lower than the installed build needs --force")
	force := fs.Bool("force", false, "reinstall the current build, allow a downgrade, or replace a dev (source-built) install")
	ref := fs.String("ref", "", "developer: build from this git ref (branch, tag, SHA) on this machine instead of using a release")
	skipImages := fs.Bool("skip-images", false, "developer (--ref only): skip the local Docker image build")
	skipFetch := fs.Bool("skip-fetch", false, "developer: reuse tree at OBELISK_ROOT instead of downloading (local image build)")
	_ = fs.Parse(args)

	chosenRef := strings.TrimSpace(*ref)
	if chosenRef == "" {
		chosenRef = strings.TrimSpace(os.Getenv("OBELISK_REF"))
	}
	if n, ok := parseBuildTag(chosenRef); ok && *to == "" {
		*to = fmt.Sprintf("build-%d", n) // `--ref build-N` is a release pin
		chosenRef = ""
	}
	devMode := chosenRef != "" || *skipFetch

	meta, _, metaErr := readInstallMeta(*root)
	if metaErr != nil {
		meta = installMeta{} // never installed by this tool or legacy: treated as build 0
	}

	var target int
	if !devMode {
		var err error
		target, err = resolveRelease(releaseBaseURL(), releaseRepo(meta.Repo), *to)
		if err != nil {
			if errors.Is(err, errNoRelease) {
				fmt.Fprintln(os.Stderr, "no release has been published yet. To build from source on this machine: sudo obelisk update --ref main")
				os.Exit(exitCannotResolve)
			}
			fmt.Fprintf(os.Stderr, "cannot determine the release to install: %v\n", err)
			os.Exit(exitCannotResolve)
		}
		installed := meta.Build
		if *check {
			if target > installed {
				fmt.Printf("update available: build %d -> build %d\n", installed, target)
				os.Exit(exitUpdateAvailable)
			}
			fmt.Printf("up to date: build %d (latest %d)\n", installed, target)
			return
		}
		if meta.Source == "dev" && !*force {
			fmt.Fprintln(os.Stderr, "this install was built from source (dev). Use --force to replace it with a release.")
			os.Exit(1)
		}
		switch {
		case target == installed && !*force:
			fmt.Printf("already at build %d\n", installed)
			return
		case target < installed && !*force:
			if *to == "" {
				fmt.Printf("already at build %d (latest release is %d)\n", installed, target)
				return
			}
			fmt.Fprintf(os.Stderr, "build %d is older than the installed build %d. Downgrades can fail if state moved forward; re-run with --force to proceed.\n", target, installed)
			os.Exit(1)
		}
		fmt.Printf("updating build %d -> build %d\n", installed, target)
	}

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "obelisk update must run as root (sudo obelisk update)")
		os.Exit(1)
	}

	script := filepath.Join(*root, "install", "update.sh")
	if _, err := os.Stat(script); err != nil {
		bootstrap := filepath.Join(*root, "install", "bootstrap.sh")
		if _, err2 := os.Stat(bootstrap); err2 != nil {
			fmt.Fprintf(os.Stderr, "no install tree at %s (missing install/update.sh)\n", *root)
			os.Exit(1)
		}
		fmt.Println("upgrade script not found; running bootstrap only")
		runBootstrapOnly(*root, *skipImages)
		return
	}

	env := append(os.Environ(),
		"OBELISK_ROOT="+*root,
		"OBELISK_INSTALL_UPGRADE=true",
	)
	if devMode {
		if chosenRef == "" {
			chosenRef = "main"
		}
		env = append(env, "OBELISK_REF="+chosenRef, "OBELISK_SOURCE=dev")
		if *skipImages {
			env = append(env, "OBELISK_BUILD_IMAGES=false")
		}
		if *skipFetch {
			env = append(env, "OBELISK_SKIP_SOURCE_FETCH=true")
		}
	} else {
		env = append(env,
			fmt.Sprintf("OBELISK_REF=build-%d", target),
			"OBELISK_SOURCE=release",
			"OBELISK_REPO="+releaseRepo(meta.Repo),
		)
	}

	cmd := exec.Command("bash", script)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = *root
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveObeliskRoot() string {
	if r := strings.TrimSpace(os.Getenv("OBELISK_ROOT")); r != "" {
		return r
	}
	return "/opt/obelisk"
}

func runBootstrapOnly(root string, skipImages bool) {
	script := filepath.Join(root, "install", "bootstrap.sh")
	env := append(os.Environ(), "OBELISK_ROOT="+root, "OBELISK_PROFILE=vps")
	if skipImages {
		env = append(env, "OBELISK_BUILD_IMAGES=false")
	} else {
		env = append(env, "OBELISK_BUILD_IMAGES=true")
	}
	if os.Getenv("OBELISK_HELM_EXTRA_ARGS") == "" {
		env = append(env, `OBELISK_HELM_EXTRA_ARGS=--set kong.service.type=LoadBalancer`)
	}
	cmd := exec.Command("bash", script)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
