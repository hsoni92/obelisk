package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type helmUpgradeOpts struct {
	waitTimeout time.Duration
}

// helmUpgradeInstallOnce runs a single helm upgrade attempt (retries/backoff live in the scheduler).
func (r *Reconciler) helmUpgradeInstallOnce(ctx context.Context, service, chart, release, ns, valuesFile string, opts helmUpgradeOpts) error {
	if opts.waitTimeout <= 0 {
		opts.waitTimeout = 2 * time.Minute
	}
	if err := r.healHelmPending(ctx, release); err != nil {
		slog.Warn("helm pending heal", "release", release, "err", err)
	}
	args := []string{"upgrade", "--install", release, chart,
		"--namespace", ns, "--create-namespace",
		"--wait", "--timeout", opts.waitTimeout.String()}
	if valuesFile != "" {
		args = append(args, "-f", valuesFile)
	}
	st, _ := r.helmReleaseStatus(ctx, release)
	if st == "pending-upgrade" || st == "pending-install" || st == "pending-rollback" {
		args = append(args, "--force")
	}
	out, err := commandRunner(ctx, "helm", args...)
	text := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("helm %s: %w: %s", release, err, text)
	}
	return nil
}

func (r *Reconciler) helmUpgradeInstall(ctx context.Context, service, chart, release, ns, valuesFile string, opts helmUpgradeOpts) error {
	if opts.waitTimeout <= 0 {
		opts.waitTimeout = 5 * time.Minute
	}
	if err := r.healHelmPending(ctx, release); err != nil {
		slog.Warn("helm pending heal", "release", release, "err", err)
	}
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
		if err := r.healHelmPending(ctx, release); err != nil {
			slog.Warn("helm pending heal", "release", release, "err", err)
		}
		args := []string{"upgrade", "--install", release, chart,
			"--namespace", ns, "--create-namespace",
			"--wait", "--timeout", opts.waitTimeout.String()}
		if valuesFile != "" {
			args = append(args, "-f", valuesFile)
		}
		st, _ := r.helmReleaseStatus(ctx, release)
		if st == "pending-upgrade" || st == "pending-install" || st == "pending-rollback" {
			args = append(args, "--force")
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		cmd.Env = kubeconfigEnv()
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("helm %s: %w: %s", release, err, text)
		lower := strings.ToLower(text)
		retry := strings.Contains(lower, "another operation") ||
			strings.Contains(lower, "pending-install") ||
			strings.Contains(lower, "pending-upgrade") ||
			strings.Contains(lower, "pending-rollback")
		if !retry {
			return lastErr
		}
	}
	return lastErr
}

func (r *Reconciler) healHelmPending(ctx context.Context, release string) error {
	st, err := r.helmReleaseStatus(ctx, release)
	if err != nil || st == "" {
		return nil
	}
	switch st {
	case "pending-upgrade", "pending-install", "pending-rollback":
		rev, err := r.lastDeployedRevision(ctx, release)
		if err != nil {
			return err
		}
		if rev > 0 {
			ns := r.namespace()
			cmd := exec.CommandContext(ctx, "helm", "rollback", release, fmt.Sprint(rev),
				"-n", ns, "--wait", "--timeout", "2m")
			cmd.Env = kubeconfigEnv()
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("rollback %s rev %d: %w: %s", release, rev, err, string(out))
			}
		}
	}
	return nil
}

func (r *Reconciler) lastDeployedRevision(ctx context.Context, release string) (int, error) {
	ns := r.namespace()
	cmd := exec.CommandContext(ctx, "helm", "history", release, "-n", ns, "-o", "json")
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
		return 0, nil
	}
	var rows []struct {
		Revision int    `json:"revision"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return 0, err
	}
	best := 0
	for _, row := range rows {
		if strings.EqualFold(row.Status, "deployed") && row.Revision > best {
			best = row.Revision
		}
	}
	return best, nil
}
