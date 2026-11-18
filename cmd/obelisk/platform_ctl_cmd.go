package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdStart(args []string)   { runPlatformCtl("start", args) }
func cmdStop(args []string)    { runPlatformCtl("stop", args) }
func cmdRestart(args []string) { runPlatformCtl("restart", args) }

func runPlatformCtl(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	root := fs.String("root", resolveObeliskRoot(), "obelisk install tree (default /opt/obelisk or OBELISK_ROOT)")
	namespace := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "kubernetes namespace")
	noWait := fs.Bool("no-wait", false, "after start/restart, do not wait for platform rollouts or Kong health")
	system := fs.Bool("system", false, "whole-node control: systemctl k3s start|stop|restart (affects every namespace)")
	_ = fs.Parse(args)

	// Both paths need root: the k3s kubeconfig is root-owned, and --system
	// manipulates the k3s service directly.
	if os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, "obelisk %s must run as root (sudo obelisk %s)\n", action, action)
		os.Exit(1)
	}

	script := filepath.Join(*root, "install", "platform-ctl.sh")
	if _, err := os.Stat(script); err != nil {
		fmt.Fprintf(os.Stderr, "missing %s — sync /opt/obelisk or set --root\n", script)
		os.Exit(1)
	}

	ctlArgs := []string{action}
	if *system {
		ctlArgs = append(ctlArgs, "--system")
	}
	if *noWait {
		ctlArgs = append(ctlArgs, "--no-wait")
	}

	env := append(os.Environ(),
		"OBELISK_ROOT="+*root,
		"OBELISK_NAMESPACE="+*namespace,
	)

	cmd := exec.Command("bash", append([]string{script}, ctlArgs...)...)
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
