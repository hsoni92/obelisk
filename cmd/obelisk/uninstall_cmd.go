package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "print planned actions only")
	yes := fs.Bool("yes", false, "skip interactive confirmation")
	tier := fs.String("tier", "full", "full (default) or platform (keep PVCs and /var/lib/obelisk)")
	keepData := fs.Bool("keep-data", false, "skip PVC deletion and keep /var/lib/obelisk")
	removeK3s := fs.Bool("remove-k3s", false, "remove k3s after Obelisk cleanup (extra confirm)")
	root := fs.String("root", resolveObeliskRoot(), "install tree")
	namespace := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "kubernetes namespace")
	force := fs.Bool("force", false, "allow uninstall when config profile is local")
	resume := fs.Bool("resume", false, "continue from last completed stage")
	_ = fs.Parse(args)

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "obelisk uninstall must run as root (sudo obelisk uninstall)")
		os.Exit(1)
	}

	script := filepath.Join(*root, "install", "uninstall.sh")
	if _, err := os.Stat(script); err != nil {
		fmt.Fprintf(os.Stderr, "missing %s — sync /opt/obelisk or set --root\n", script)
		os.Exit(1)
	}

	scriptArgs := []string{script}
	if *dryRun {
		scriptArgs = append(scriptArgs, "--dry-run")
	}
	if *yes {
		scriptArgs = append(scriptArgs, "--yes")
	}
	t := strings.TrimSpace(*tier)
	if t != "" && t != "full" {
		scriptArgs = append(scriptArgs, "--tier", t)
	}
	if *keepData {
		scriptArgs = append(scriptArgs, "--keep-data")
	}
	if *removeK3s {
		scriptArgs = append(scriptArgs, "--remove-k3s")
	}
	if *force {
		scriptArgs = append(scriptArgs, "--force")
	}
	if *resume {
		scriptArgs = append(scriptArgs, "--resume")
	}
	scriptArgs = append(scriptArgs, "--root", *root, "--namespace", *namespace)

	env := append(os.Environ(),
		"OBELISK_ROOT="+*root,
		"OBELISK_NAMESPACE="+*namespace,
	)
	if *dryRun {
		env = append(env, "OBELISK_UNINSTALL_DRY_RUN=true")
	}
	if *yes {
		env = append(env, "OBELISK_UNINSTALL_YES=true")
	}
	if *removeK3s {
		env = append(env, "OBELISK_REMOVE_K3S=true")
	}

	cmd := exec.Command("bash", scriptArgs...)
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
