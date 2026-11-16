package backup

import (
	"context"
	"fmt"
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = kubeconfigEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
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