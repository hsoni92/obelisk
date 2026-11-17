package exposure

import (
	"context"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Reconcile applies Kong routes and optional Tailscale Funnel settings.
func Reconcile(ctx context.Context, spec obelisk.Spec, helmRoot, namespace string) error {
	if err := ReconcileKong(ctx, spec, helmRoot, namespace); err != nil {
		return err
	}
	return ApplyFunnel(spec)
}