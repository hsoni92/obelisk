package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// pruneOrphanReleases uninstalls Helm releases in the namespace that no longer
// appear in the spec. Orphans are common after blueprint import when instance
// ids were remapped and reconcile targeted new release names.
func (r *Reconciler) pruneOrphanReleases(ctx context.Context, spec obelisk.Spec) []string {
	if r.reconcileDisabled() {
		return nil
	}
	desired := map[string]struct{}{
		PlatformRelease: {},
	}
	for _, inst := range spec.Instances {
		desired[inst.ReleaseName()] = struct{}{}
	}
	releases, err := r.listHelmReleases(ctx, true)
	if err != nil {
		slog.Warn("list helm releases", "err", err)
		return nil
	}
	var removed []string
	for _, rel := range releases {
		if !strings.HasPrefix(rel, "obelisk-") {
			continue
		}
		if _, ok := desired[rel]; ok {
			continue
		}
		if err := r.helmUninstall(ctx, rel); err != nil {
			slog.Warn("prune orphan release", "release", rel, "err", err)
			continue
		}
		removed = append(removed, rel)
	}
	return removed
}

func (r *Reconciler) listHelmReleases(ctx context.Context, all bool) ([]string, error) {
	ns := r.namespace()
	args := []string{"list", "-n", ns, "-q"}
	if all {
		args = append(args, "-a")
	}
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Env = kubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	if len(names) > 0 {
		return names, nil
	}
	// Some Helm builds emit JSON when -o json is used; fall back to table parse only.
	args = []string{"list", "-n", ns, "-o", "json"}
	if all {
		args = append(args, "-a")
	}
	cmd = exec.CommandContext(ctx, "helm", args...)
	cmd.Env = kubeconfigEnv()
	raw, err = cmd.Output()
	if err != nil {
		return nil, err
	}
	var docs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &docs); err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.Name != "" {
			names = append(names, d.Name)
		}
	}
	return names, nil
}

func (r *Reconciler) helmUninstall(ctx context.Context, release string) error {
	ns := r.namespace()
	cmd := exec.CommandContext(ctx, "helm", "uninstall", release, "-n", ns)
	cmd.Env = kubeconfigEnv()
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if strings.Contains(text, "not found") {
			return nil
		}
		return fmt.Errorf("helm uninstall %s: %w: %s", release, err, text)
	}
	slog.Info("pruned orphan release", "release", release)
	return nil
}