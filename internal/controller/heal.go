package controller

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const stuckPendingInstallAge = 6 * time.Minute

func backoffDuration(attempts int) time.Duration {
	switch {
	case attempts <= 1:
		return 30 * time.Second
	case attempts == 2:
		return time.Minute
	case attempts == 3:
		return 2 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func (r *Reconciler) safeHealHelm(ctx context.Context, release, helmStatus string, pendingSince time.Time) error {
	switch helmStatus {
	case "pending-upgrade", "pending-rollback", "failed":
		rev, err := r.lastDeployedRevision(ctx, release)
		if err != nil {
			return err
		}
		if rev > 0 {
			ns := r.namespace()
			out, err := commandRunner(ctx, "helm", "rollback", release, fmt.Sprint(rev),
				"-n", ns, "--wait", "--timeout", "2m")
			if err != nil {
				return fmt.Errorf("rollback %s rev %d: %w: %s", release, rev, err, strings.TrimSpace(string(out)))
			}
			return nil
		}
	case "pending-install":
		if !pendingSince.IsZero() && time.Since(pendingSince) >= stuckPendingInstallAge {
			rev, _ := r.lastDeployedRevision(ctx, release)
			if rev == 0 {
				ns := r.namespace()
				out, err := commandRunner(ctx, "helm", "uninstall", release, "-n", ns)
				if err != nil {
					text := strings.TrimSpace(string(out))
					if !strings.Contains(strings.ToLower(text), "not found") {
						return fmt.Errorf("clear stuck pending install %s: %w: %s", release, err, text)
					}
				}
				slog.Info("cleared stuck pending-install release record", "release", release)
			}
		}
	}
	return nil
}

func helmErrorRetryable(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "another operation") ||
		strings.Contains(lower, "pending-install") ||
		strings.Contains(lower, "pending-upgrade") ||
		strings.Contains(lower, "pending-rollback") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded")
}