package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func cmdGateway(args []string) {
	if len(args) < 1 {
		gatewayUsage()
		os.Exit(1)
	}
	reexecWithSudoForK3s()
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "enable":
		cmdGatewayEnable(args[1:])
	case "local":
		cmdGatewayLocal(args[1:])
	case "status":
		cmdGatewayStatus(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown gateway command %q (try: enable, local, status)\n", args[0])
		gatewayUsage()
		os.Exit(1)
	}
}

func gatewayUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  obelisk gateway enable [--profile vps|local] [--config path] [--no-reconcile]")
	fmt.Fprintln(os.Stderr, "  obelisk gateway local  [--config path] [--no-reconcile]")
	fmt.Fprintln(os.Stderr, "  obelisk gateway status [--config path]")
}

func cmdGatewayEnable(args []string) {
	fs := flag.NewFlagSet("gateway enable", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	profile := fs.String("profile", "", "optional: set deployment.profile to vps or local")
	helmRoot := fs.String("helm-root", config.ResolveHelmRoot(), "helm charts root")
	namespace := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "kubernetes namespace")
	doReconcile := fs.Bool("reconcile", true, "reconcile after save")
	publicHost := fs.String("public-host", "", "set controller OBELISK_PUBLIC_HOST")
	_ = fs.Parse(args)

	spec, err := config.Load(*cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if p := strings.ToLower(strings.TrimSpace(*profile)); p != "" {
		switch p {
		case "vps", "local":
			spec.Deployment.Profile = p
		default:
			fmt.Fprintln(os.Stderr, "profile must be vps or local")
			os.Exit(1)
		}
	}
	spec.Deployment.Role = obelisk.DeploymentRoleGateway
	spec.Deployment.Tailscale.Enabled = true
	obelisk.NormalizeDeployment(&spec)
	obelisk.NormalizePeerGateway(&spec)

	if err := config.Validate(&spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := config.Save(*cfg, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	wrote := config.WriteConfigPath(*cfg)
	fmt.Printf("gateway enabled: profile=%s role=%s tailscale=true (wrote %s)\n",
		spec.Deployment.Profile, spec.Deployment.Role, wrote)
	fmt.Println("on the host run: sudo obelisk tailscale install   then join from Platform → Tailscale or: tailscale up")

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

func cmdGatewayLocal(args []string) {
	fs := flag.NewFlagSet("gateway local", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	helmRoot := fs.String("helm-root", config.ResolveHelmRoot(), "helm charts root")
	namespace := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "kubernetes namespace")
	doReconcile := fs.Bool("reconcile", true, "reconcile after save")
	_ = fs.Parse(args)

	spec, err := config.Load(*cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spec.Deployment.Role = obelisk.DeploymentRoleLocal
	if spec.Deployment.Profile == "vps" {
		spec.Deployment.Tailscale.Enabled = false
	}
	obelisk.NormalizeDeployment(&spec)
	if err := config.Validate(&spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := config.Save(*cfg, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	wrote := config.WriteConfigPath(*cfg)
	fmt.Printf("gateway disabled: role=local profile=%s tailscale=%v (wrote %s)\n",
		spec.Deployment.Profile, spec.Deployment.Tailscale.Enabled, wrote)

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

func cmdGatewayStatus(args []string) {
	fs := flag.NewFlagSet("gateway status", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	_ = fs.Parse(args)
	spec, err := config.Load(*cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("profile=%s role=%s tailscale=%v\n",
		spec.Deployment.Profile, spec.Deployment.Role, spec.Deployment.Tailscale.Enabled)
	fmt.Printf("config=%s\n", config.WriteConfigPath(*cfg))
	if obelisk.IsGateway(spec) {
		fmt.Printf("peer_gateway.auto_admit=%v tls_mode=%s\n",
			obelisk.PeerGatewayAutoAdmit(spec), obelisk.PeerGatewayTLSMode(spec))
	}
}
