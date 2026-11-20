package tailscale

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func needsTailscaleElevation(out []byte, err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error() + " " + string(out))
	return strings.Contains(msg, "access denied") ||
		strings.Contains(msg, "checkprefs") ||
		strings.Contains(msg, "must be root") ||
		strings.Contains(msg, "use 'sudo tailscale")
}

func sudoNeedsPassword(out []byte, err error) bool {
	msg := strings.ToLower(err.Error() + " " + string(out))
	return strings.Contains(msg, "password is required") ||
		strings.Contains(msg, "a terminal is required")
}

func tailscaleLocalSetupHint() string {
	name := strings.TrimSpace(os.Getenv("USER"))
	if u, e := user.Current(); e == nil && strings.TrimSpace(u.Username) != "" {
		name = u.Username
	}
	if name == "" {
		name = "YOUR_USER"
	}
	return fmt.Sprintf(
		"run once in a terminal (sudo password): sudo tailscale set --operator=%s — then retry Connect tailnet",
		name,
	)
}

// hostNsenterUP is only valid as root (controller pod). Unprivileged local dev cannot use nsenter.
func hostNsenterUP() bool {
	return os.Geteuid() == 0
}

func runCombined(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// cliPrivileged runs tailscale on the host when the caller lacks checkprefs (typical for
// the controller pod or a dev API started as a non-root user).
func cliPrivileged(ctx context.Context, args ...string) ([]byte, error) {
	env := tailscaleEnv()

	var (
		lastErr             error
		sudoPasswordBlocked bool
	)
	if _, err := exec.LookPath("sudo"); err == nil {
		sudoArgs := append([]string{"-n", "env", "TS_SOCKET=" + socketPath()}, append([]string{"tailscale"}, args...)...)
		out, err := runCombined(ctx, "sudo", sudoArgs, env)
		if err == nil {
			return out, nil
		}
		if sudoNeedsPassword(out, err) {
			sudoPasswordBlocked = true
			lastErr = err
		} else if !needsTailscaleElevation(out, err) {
			return out, err
		} else {
			lastErr = err
		}
	}

	if hostNsenterUP() {
		if _, err := exec.LookPath("nsenter"); err == nil {
			hostArgs := append(
				[]string{"-t", "1", "-m", "-u", "-n", "-i", "-p", "--", "tailscale"},
				args...,
			)
			out, err := runCombined(ctx, "nsenter", hostArgs, env)
			if err == nil {
				return out, nil
			}
			if !needsTailscaleElevation(out, err) && !sudoNeedsPassword(out, err) {
				return out, err
			}
			lastErr = err
		}
	}

	if sudoPasswordBlocked || !hostNsenterUP() {
		return nil, fmt.Errorf("%s", tailscaleLocalSetupHint())
	}
	if lastErr != nil {
		return nil, fmt.Errorf("%w; %s", lastErr, tailscaleLocalSetupHint())
	}
	return nil, fmt.Errorf("tailscale needs host privileges; %s", tailscaleLocalSetupHint())
}