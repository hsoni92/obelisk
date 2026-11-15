package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// K8sDataBackupEnabled is true when kubectl can exec into data service pods.
func K8sDataBackupEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OBELISK_RECONCILE"))) {
	case "simulate", "sim", "off", "dry-run":
		return false
	case "cluster":
		return kubectlAvailable()
	}
	if strings.TrimSpace(os.Getenv("OBELISK_DEV")) == "1" {
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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return out, nil
}