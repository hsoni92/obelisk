package main

import (
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
	ref := fs.String("ref", "", "GitHub archive ref (branch, tag, SHA); default main or prior .install-meta.json")
	skipImages := fs.Bool("skip-images", false, "skip Docker image rebuild (helm upgrade only)")
	skipFetch := fs.Bool("skip-fetch", false, "reuse tree at OBELISK_ROOT (bootstrap / image build only)")
	_ = fs.Parse(args)

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "obelisk update must run as root on a VPS (sudo obelisk update)")
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

	chosenRef := strings.TrimSpace(*ref)
	if chosenRef == "" {
		chosenRef = strings.TrimSpace(os.Getenv("OBELISK_REF"))
	}
	if chosenRef == "" {
		chosenRef = "main"
	}

	env := append(os.Environ(),
		"OBELISK_ROOT="+*root,
		"OBELISK_REF="+chosenRef,
		"OBELISK_INSTALL_UPGRADE=true",
	)
	if *skipImages {
		env = append(env, "OBELISK_BUILD_IMAGES=false")
	}
	if *skipFetch {
		env = append(env, "OBELISK_SKIP_SOURCE_FETCH=true")
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
