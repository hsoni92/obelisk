package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/tailscale"
)

func cmdTailscale(args []string) {
	if len(args) < 1 {
		tailscaleUsage()
		os.Exit(1)
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "install", "ensure":
		cmdTailscaleInstall(args[1:])
	case "status":
		cmdTailscaleStatus(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown tailscale command %q (try: install, status)\n", args[0])
		tailscaleUsage()
		os.Exit(1)
	}
}

func tailscaleUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  obelisk tailscale install [--dry-run]   detect OS and install/start tailscaled on the host")
	fmt.Fprintln(os.Stderr, "  obelisk tailscale ensure [--dry-run]    alias for install")
	fmt.Fprintln(os.Stderr, "  obelisk tailscale status [--json]")
}

func cmdTailscaleInstall(args []string) {
	reexecWithSudoForTailscale()
	fs := flag.NewFlagSet("tailscale install", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "print steps without changing the system")
	_ = fs.Parse(args)

	p := tailscale.DetectPlatform()
	fmt.Printf("host: os=%s", p.OS)
	if p.ID != "" {
		fmt.Printf(" distro=%s", p.ID)
	}
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := tailscale.EnsureHost(ctx, *dryRun); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *dryRun {
		fmt.Println("dry-run complete")
		return
	}
	st := tailscale.GetStatus(ctx)
	if st.Error != "" && !st.Installed {
		fmt.Fprintln(os.Stderr, st.Error)
		os.Exit(1)
	}
	fmt.Println("tailscale host ready; join with: obelisk gateway enable  or  Platform → Tailscale")
	if st.DNSName != "" {
		fmt.Printf("node: %s\n", strings.TrimSuffix(st.DNSName, "."))
	}
}

func cmdTailscaleStatus(args []string) {
	fs := flag.NewFlagSet("tailscale status", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit JSON")
	_ = fs.Parse(args)
	st := tailscale.GetStatus(context.Background())
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(st)
		return
	}
	fmt.Printf("installed=%v socket=%q backend=%s connected=%v\n",
		st.Installed, st.SocketPath, st.BackendState, st.Connected)
	if st.DNSName != "" {
		fmt.Printf("dns_name=%s\n", strings.TrimSuffix(st.DNSName, "."))
	}
	if st.Error != "" {
		fmt.Printf("error: %s\n", st.Error)
	}
}

// reexecWithSudoForTailscale runs install/ensure as root when not already root.
func reexecWithSudoForTailscale() {
	if os.Geteuid() == 0 {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot find obelisk binary:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "tailscale install needs root; re-running with sudo")
	cmd := exec.Command("sudo", append([]string{"--preserve-env", "--", exe}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
