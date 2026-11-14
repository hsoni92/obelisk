package controller

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (r *Reconciler) helmUpgradeInstall(ctx context.Context, service, chart, release, ns, valuesFile string) error {
	deadline := time.Now().Add(8 * time.Minute)
	var lastErr error
	for attempt := 0; time.Now().Before(deadline); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		args := []string{"upgrade", "--install", release, chart,
			"--namespace", ns, "--create-namespace",
			"--wait", "--timeout", "5m"}
		if valuesFile != "" {
			args = append(args, "-f", valuesFile)
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		cmd.Env = kubeconfigEnv()
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("helm %s: %w: %s", release, err, text)
		if !strings.Contains(text, "another operation") && !strings.Contains(text, "pending-install") {
			return lastErr
		}
	}
	return lastErr
}