package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/controller"
)

// K8sDataBackupEnabled is true when kubectl can exec into data service pods.
func K8sDataBackupEnabled() bool {
	if controller.ReconcileDisabled() {
		return false
	}
	return kubectlAvailable()
}

func kubectlAvailable() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

func obeliskNamespace() string {
	if ns := strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE")); ns != "" {
		return ns
	}
	return "obelisk"
}

func kubectlOutput(ctx context.Context, args ...string) ([]byte, error) {
	out, err := kubectlStdout(ctx, args...)
	if err != nil {
		stderr := ""
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		msg := strings.Join(args, " ")
		if stderr != "" {
			return nil, fmt.Errorf("kubectl %s: %w (%s)", msg, err, stderr)
		}
		return nil, fmt.Errorf("kubectl %s: %w", msg, err)
	}
	return out, nil
}

// kubectlCombined runs kubectl with optional stdin; returns combined output on failure.
func kubectlCombined(ctx context.Context, stdin io.Reader, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = kubeconfigEnv()
	if stdin != nil {
		cmd.Stdin = stdin
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		stderr := strings.TrimSpace(string(out))
		msg := strings.Join(args, " ")
		if stderr != "" {
			return nil, fmt.Errorf("kubectl %s: %w (%s)", msg, err, stderr)
		}
		return nil, fmt.Errorf("kubectl %s: %w", msg, err)
	}
	return out, nil
}

// kubectlStdout captures stdout only (safe for binary dumps such as tar).
func kubectlStdout(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = kubeconfigEnv()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func kubeconfigEnv() []string {
	// Reuse controller kubeconfig resolution via small duplicate to avoid export cycle.
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("KUBECONFIG")) != "" {
		return env
	}
	if strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")) != "" {
		return env
	}
	if _, err := os.Stat("/etc/rancher/k3s/k3s.yaml"); err == nil {
		return append(env, "KUBECONFIG=/etc/rancher/k3s/k3s.yaml")
	}
	return env
}
