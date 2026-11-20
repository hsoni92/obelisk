package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Status is a reduced view of `tailscale status --json`.
type Status struct {
	Installed   bool     `json:"installed"`
	BackendOK   bool     `json:"backend_ok"`
	Connected   bool     `json:"connected"`
	BackendState string  `json:"backend_state,omitempty"`
	AuthURL     string   `json:"auth_url,omitempty"`
	Tailnet     string   `json:"tailnet,omitempty"`
	DNSName     string   `json:"dns_name,omitempty"`
	IPs         []string `json:"ips,omitempty"`
	SocketPath  string   `json:"socket_path"`
	Error       string   `json:"error,omitempty"`
}

type statusJSON struct {
	BackendState string `json:"BackendState"`
	AuthURL      string `json:"AuthURL"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	Self         struct {
		DNSName string `json:"DNSName"`
	} `json:"Self"`
	CurrentTailnet struct {
		Name string `json:"Name"`
	} `json:"CurrentTailnet"`
}

func socketPath() string {
	if p := strings.TrimSpace(os.Getenv("OBELISK_TAILSCALE_SOCKET")); p != "" {
		return p
	}
	return "/var/run/tailscale/tailscaled.sock"
}

func tailscaleEnv() []string {
	env := os.Environ()
	env = append(env, "TS_SOCKET="+socketPath())
	return env
}

func cli(ctx context.Context, args ...string) ([]byte, error) {
	return runCombined(ctx, "tailscale", args, tailscaleEnv())
}

// CLIInstalled reports whether the tailscale binary is on PATH.
func CLIInstalled() bool {
	_, err := exec.LookPath("tailscale")
	return err == nil
}

// SocketPresent reports whether the host tailscaled socket exists.
func SocketPresent() bool {
	_, err := os.Stat(socketPath())
	return err == nil
}

// GetStatus queries tailscaled via the CLI.
func GetStatus(ctx context.Context) Status {
	st := Status{
		Installed:  CLIInstalled(),
		SocketPath: socketPath(),
	}
	if !st.Installed {
		st.Error = "tailscale CLI not installed on this host"
		return st
	}
	if !SocketPresent() {
		st.Error = "tailscaled socket not found; install and start tailscaled on the host"
		return st
	}
	out, err := cli(ctx, "status", "--json")
	if err != nil {
		st.Error = err.Error()
		return st
	}
	var raw statusJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		st.Error = "parse tailscale status: " + err.Error()
		return st
	}
	st.BackendState = raw.BackendState
	st.BackendOK = strings.EqualFold(raw.BackendState, "Running")
	st.AuthURL = strings.TrimSpace(raw.AuthURL)
	st.DNSName = strings.TrimSpace(raw.Self.DNSName)
	st.Tailnet = strings.TrimSpace(raw.CurrentTailnet.Name)
	st.IPs = append([]string(nil), raw.TailscaleIPs...)
	st.Connected = st.BackendOK && len(st.IPs) > 0 && st.AuthURL == ""
	return st
}

// Up connects the node with a one-time or reusable auth key (never logged).
func Up(ctx context.Context, authKey string) error {
	authKey = strings.TrimSpace(authKey)
	if authKey == "" {
		return fmt.Errorf("auth key required")
	}
	if !CLIInstalled() {
		return fmt.Errorf("tailscale CLI not installed")
	}
	if !SocketPresent() {
		return fmt.Errorf("tailscaled socket not found")
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	args := []string{"up", "--auth-key=" + authKey, "--reset=false"}
	out, err := cli(cctx, args...)
	if err == nil {
		return nil
	}
	if !needsTailscaleElevation(out, err) {
		return err
	}
	_, err = cliPrivileged(cctx, args...)
	return err
}