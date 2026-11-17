package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
)

func cmdConfigureDeployment(args []string) {
	fs := flag.NewFlagSet("configure deployment", flag.ExitOnError)
	profile := fs.String("profile", "", "deployment profile: vps or local")
	tailscale := fs.Bool("tailscale", false, "enable Tailscale (local profile only)")
	show := fs.Bool("show", false, "print current deployment profile and exit")
	cfg := fs.String("config", config.ResolvePath(), "config path")
	helmRoot := fs.String("helm-root", config.ResolveHelmRoot(), "helm charts root")
	doReconcile := fs.Bool("reconcile", true, "reconcile services and Kong exposure after save")
	publicHost := fs.String("public-host", "", "set controller OBELISK_PUBLIC_HOST (VPS connection hints)")
	namespace := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "kubernetes namespace")
	_ = fs.Parse(args)

	spec, err := config.Load(*cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *show {
		fmt.Printf("profile=%s tailscale=%v\n", spec.Deployment.Profile, spec.Deployment.Tailscale.Enabled)
		return
	}

	if *profile == "" {
		fmt.Fprintln(os.Stderr, "usage: obelisk configure deployment --profile vps|local")
		fmt.Fprintln(os.Stderr, "       obelisk configure deployment --show")
		os.Exit(1)
	}

	switch strings.ToLower(strings.TrimSpace(*profile)) {
	case "vps":
		spec.Deployment.Profile = "vps"
		spec.Deployment.Tailscale.Enabled = false
	case "local":
		spec.Deployment.Profile = "local"
		spec.Deployment.Tailscale.Enabled = *tailscale
	default:
		fmt.Fprintln(os.Stderr, "profile must be vps or local")
		os.Exit(1)
	}

	if err := config.Save(*cfg, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	wrote := config.WriteConfigPath(*cfg)
	fmt.Printf("deployment.profile=%s (wrote %s)\n", spec.Deployment.Profile, wrote)

	if ph := strings.TrimSpace(*publicHost); ph != "" {
		if err := setControllerPublicHost(ph, *namespace); err != nil {
			fmt.Fprintln(os.Stderr, "public-host:", err)
			os.Exit(1)
		}
		fmt.Printf("OBELISK_PUBLIC_HOST=%s on obelisk-controller\n", ph)
	}

	if *doReconcile {
		rec := controller.Reconciler{ConfigPath: *cfg, HelmRoot: *helmRoot, Namespace: *namespace}
		rep, err := rec.Reconcile(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
	}
}

func setControllerPublicHost(host, namespace string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("empty host")
	}
	cmd := exec.Command("kubectl", "set", "env", "deployment/obelisk-controller",
		"-n", namespace, "OBELISK_PUBLIC_HOST="+host)
	cmd.Env = kubeconfigEnvForCLI()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	roll := exec.Command("kubectl", "rollout", "status", "deployment/obelisk-controller",
		"-n", namespace, "--timeout=120s")
	roll.Env = kubeconfigEnvForCLI()
	if out, err := roll.CombinedOutput(); err != nil {
		return fmt.Errorf("controller rollout: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func kubeconfigEnvForCLI() []string {
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("KUBECONFIG")) != "" {
		return env
	}
	if _, err := os.Stat("/etc/rancher/k3s/k3s.yaml"); err == nil {
		return append(env, "KUBECONFIG=/etc/rancher/k3s/k3s.yaml")
	}
	return env
}