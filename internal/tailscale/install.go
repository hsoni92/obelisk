package tailscale

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Platform summarizes the host OS for install routing.
type Platform struct {
	OS     string // linux, darwin, …
	ID     string // os-release ID on Linux
	IDLike string // os-release ID_LIKE on Linux
}

// DetectPlatform reads runtime.GOOS and, on Linux, /etc/os-release.
func DetectPlatform() Platform {
	p := Platform{OS: runtime.GOOS}
	if runtime.GOOS != "linux" {
		return p
	}
	lp, err := parseOSReleaseFile("/etc/os-release")
	if err != nil {
		return p
	}
	lp.OS = p.OS
	return lp
}

func parseOSReleaseFile(path string) (Platform, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Platform{}, err
	}
	p := Platform{}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"`)
		switch k {
		case "ID":
			p.ID = strings.ToLower(v)
		case "ID_LIKE":
			p.IDLike = strings.ToLower(v)
		}
	}
	return p, nil
}

// EnsureHost installs Tailscale when missing and starts tailscaled when the socket is absent.
// It is idempotent. dryRun prints planned steps without changing the system.
func EnsureHost(ctx context.Context, dryRun bool) error {
	if SocketPresent() && CLIInstalled() {
		return nil
	}
	if CLIInstalled() {
		return startDaemon(ctx, dryRun)
	}
	if err := installPackage(ctx, dryRun); err != nil {
		return err
	}
	return startDaemon(ctx, dryRun)
}

func installPackage(ctx context.Context, dryRun bool) error {
	switch runtime.GOOS {
	case "linux":
		return installLinux(ctx, dryRun)
	case "darwin":
		return installDarwin(ctx, dryRun)
	default:
		return fmt.Errorf("tailscale auto-install is not supported on %s (use https://tailscale.com/download)", runtime.GOOS)
	}
}

func installLinux(ctx context.Context, dryRun bool) error {
	if _, err := exec.LookPath("curl"); err != nil {
		return fmt.Errorf("curl not found; install curl or use your distro package manager for tailscale")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		return fmt.Errorf("sh not found")
	}
	cmdLine := "curl -fsSL https://tailscale.com/install.sh | sh"
	if dryRun {
		fmt.Println("[dry-run]", cmdLine)
		return nil
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdLine)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		p := DetectPlatform()
		return fmt.Errorf("tailscale install failed (%s): %w — try https://tailscale.com/download/linux", distroLabel(p), err)
	}
	return nil
}

func installDarwin(ctx context.Context, dryRun bool) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("Homebrew not found; install Tailscale from https://tailscale.com/download/mac or: brew install tailscale")
	}
	return runLogged(ctx, dryRun, "install tailscale via Homebrew", "brew", "install", "tailscale")
}

func startDaemon(ctx context.Context, dryRun bool) error {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("systemctl"); err != nil {
			return fmt.Errorf("systemctl not found; start tailscaled manually")
		}
		if err := runLogged(ctx, dryRun, "enable and start tailscaled", "systemctl", "enable", "--now", "tailscaled"); err != nil {
			return err
		}
		if dryRun {
			return nil
		}
		if !SocketPresent() {
			return fmt.Errorf("tailscaled still not running; check: systemctl status tailscaled")
		}
		return nil
	case "darwin":
		if _, err := exec.LookPath("brew"); err != nil {
			return fmt.Errorf("Homebrew not found; start Tailscale from the menu bar app or install brew and run: brew services start tailscale")
		}
		if err := runLogged(ctx, dryRun, "start tailscale via brew services", "brew", "services", "start", "tailscale"); err != nil {
			return fmt.Errorf("could not start tailscale on macOS: %w (try the Tailscale app from the Mac App Store)", err)
		}
		return nil
	default:
		return fmt.Errorf("cannot start tailscaled on %s", runtime.GOOS)
	}
}

func runLogged(ctx context.Context, dryRun bool, label string, name string, args ...string) error {
	if dryRun {
		fmt.Printf("[dry-run] %s: %s %s\n", label, name, strings.Join(args, " "))
		return nil
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

func distroLabel(p Platform) string {
	if p.ID != "" {
		return p.ID
	}
	if p.OS != "" {
		return p.OS
	}
	return "linux"
}
