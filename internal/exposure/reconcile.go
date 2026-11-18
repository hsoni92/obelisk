package exposure

import (
	"context"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Reconcile applies Kong routes and optional Tailscale Funnel settings.
func Reconcile(ctx context.Context, spec obelisk.Spec, helmRoot, namespace string) error {
	if obelisk.VPSEdge(spec) && NeedsNativeEdgeTLS(spec) {
		if err := ReconcileEdgeTLSSecret(ctx, namespace); err != nil {
			return err
		}
	}
	if err := ReconcileKong(ctx, spec, helmRoot, namespace); err != nil {
		return err
	}
	if err := ReconcileNativeEdge(ctx, spec, namespace); err != nil {
		return err
	}
	return ApplyFunnel(spec)
}