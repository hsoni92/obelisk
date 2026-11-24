package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdCleanup(args []string) {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "print planned actions only")
	deep := fs.Bool("deep", false, "aggressive prune (Docker build cache; tighter Helm history)")
	root := fs.String("root", resolveObeliskRoot(), "obelisk install tree")
	namespace := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "kubernetes namespace")
	_ = fs.Parse(args)

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "obelisk cleanup must run as root on the appliance (sudo obelisk cleanup)")
		os.Exit(1)
	}

	script := filepath.Join(*root, "install", "cleanup.sh")
	if _, err := os.Stat(script); err != nil {
		fmt.Fprintf(os.Stderr, "missing %s — sync /opt/obelisk or set --root\n", script)
		os.Exit(1)
	}

	cmdArgs := []string{script}
	if *dryRun {
		cmdArgs = append(cmdArgs, "--dry-run")
	}
	if *deep {
		cmdArgs = append(cmdArgs, "--deep")
	}

	helmKeep := 10
	if *deep {
		helmKeep = 5
	}
	env := append(os.Environ(),
		"OBELISK_ROOT="+*root,
		"OBELISK_NAMESPACE="+*namespace,
		fmt.Sprintf("OBELISK_HELM_HISTORY_MAX=%d", helmKeep),
	)
	if *dryRun {
		env = append(env, "OBELISK_CLEANUP_DRY_RUN=true")
	}
	if *deep {
		env = append(env, "OBELISK_CLEANUP_DEEP=true")
	}

	cmd := exec.Command("bash", cmdArgs...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
