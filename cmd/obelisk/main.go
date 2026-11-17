package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "status":
		cmdStatus(os.Args[2:]) // human-friendly report; --json for machines
	case "preflight":
		cmdPreflight(os.Args[2:])
	case "service":
		cmdService(os.Args[2:])
	case "config", "init-config":
		cmdInitConfig(os.Args[2:])
	case "reconcile":
		cmdReconcile(os.Args[2:])
	case "blueprint":
		cmdBlueprint(os.Args[2:])
	case "restore":
		cmdRestore(os.Args[2:])
	case "github":
		cmdGitHub(os.Args[2:])
	case "configure":
		cmdConfigure(os.Args[2:])
	case "update":
		cmdUpdate(os.Args[2:])
	case "cleanup":
		cmdCleanup(os.Args[2:])
	case "start":
		cmdStart(os.Args[2:])
	case "stop":
		cmdStop(os.Args[2:])
	case "restart":
		cmdRestart(os.Args[2:])
	case "uninstall":
		cmdUninstall(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`obelisk — middleware appliance CLI

Commands:
  preflight [--profile vps|local] [--tailscale]
  init-config --profile vps|local [--path /etc/obelisk/obelisk.yaml]
  status [--json] [--config path] [--helm-root path]
  service enable|disable <postgres|mysql|mongodb|redis|rustfs|qdrant|searxng> [--config path] [--helm-root path]
  reconcile [--config path] [--helm-root path]
  blueprint status|unlock|lock|push [--api URL] [--config path] [--passphrase secret]
  restore --chain [--api URL] [--config path] [--passphrase secret] [--adopt-machine-id | --keep-machine-id]
  configure github   interactive GitHub OAuth wizard (same as github oauth configure)
  github oauth configure|set|status   blueprint login credentials (set = non-interactive flags)
  update [--ref REF] [--root /opt/obelisk] [--skip-images] [--skip-fetch]
  start|stop|restart [--root /opt/obelisk] [--namespace NAME]   VPS: k3s platform lifecycle
  cleanup [--dry-run] [--deep]
  uninstall [--dry-run] [--yes] [--tier full|platform] [--keep-data] [--remove-k3s] [--root path]
`)
}

func cmdPreflight(args []string) {
	fs := flag.NewFlagSet("preflight", flag.ExitOnError)
	profile := fs.String("profile", "vps", "deployment profile")
	tailscale := fs.Bool("tailscale", false, "enable tailscale (local only)")
	_ = fs.Parse(args)
	if *profile == "vps" && *tailscale {
		fmt.Fprintln(os.Stderr, "tailscale is only valid with --profile local")
		os.Exit(1)
	}
	fmt.Printf("preflight ok: profile=%s tailscale=%v\n", *profile, *tailscale)
}

func cmdInitConfig(args []string) {
	fs := flag.NewFlagSet("init-config", flag.ExitOnError)
	profile := fs.String("profile", "vps", "deployment profile")
	path := fs.String("path", config.ResolvePath(), "config path")
	tailscale := fs.Bool("tailscale", false, "enable tailscale (local only)")
	_ = fs.Parse(args)
	spec := obelisk.DefaultSpec(*profile)
	if *profile == "local" {
		spec.Deployment.Tailscale.Enabled = *tailscale
	}
	if err := config.Save(*path, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote", *path)
}

func cmdService(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: obelisk service enable|disable <name>")
		os.Exit(1)
	}
	action, name := args[0], args[1]
	fs := flag.NewFlagSet("service", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	helmRoot := fs.String("helm-root", config.ResolveHelmRoot(), "helm charts root")
	_ = fs.Parse(args[2:])
	spec, err := config.Load(*cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enabled := action == "enable"
	if action != "enable" && action != "disable" {
		fmt.Fprintln(os.Stderr, "action must be enable or disable")
		os.Exit(1)
	}
	if err := controller.SetServiceEnabled(&spec, name, enabled); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := config.Save(*cfg, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rec := controller.Reconciler{ConfigPath: *cfg, HelmRoot: *helmRoot}
	rep, err := rec.Reconcile(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
}

func cmdReconcile(args []string) {
	fs := flag.NewFlagSet("reconcile", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	helmRoot := fs.String("helm-root", config.ResolveHelmRoot(), "helm charts root")
	_ = fs.Parse(args)
	rec := controller.Reconciler{ConfigPath: *cfg, HelmRoot: *helmRoot}
	rep, err := rec.Reconcile(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
}
