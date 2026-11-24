package exposure

import (
	"context"

	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Reconcile applies Kong routes and optional Tailscale Funnel settings.
func Reconcile(ctx context.Context, spec obelisk.Spec, helmRoot, namespace string, federated []peer.FederatedTarget, configPath string) error {
	var backhaul *peer.BackhaulInfo
	if bh, ok := peer.LocalBackhaul(configPath, spec); ok {
		backhaul = &bh
	}
	if NeedsEdgeTLSSecret(spec, federated, backhaul) {
		domain := edgeTLSBaseDomain(spec, backhaul)
		if err := ReconcileEdgeTLSSecretDomain(ctx, namespace, domain); err != nil {
			return err
		}
	}
	if err := ReconcileKong(ctx, spec, helmRoot, namespace, federated, backhaul); err != nil {
		return err
	}
	if err := ReconcileNativeEdge(ctx, spec, namespace, federated, backhaul); err != nil {
		return err
	}
	return ApplyFunnel(spec)
}

func NeedsEdgeTLSSecret(spec obelisk.Spec, federated []peer.FederatedTarget, backhaul *peer.BackhaulInfo) bool {
	if obelisk.VPSEdge(spec) && NeedsNativeEdgeTLS(spec) {
		return true
	}
	if backhaul != nil && backhaul.GatewayEdgeDomain != "" && NeedsNativeEdgeTLS(spec) {
		return true
	}
	return len(federated) > 0 && obelisk.ConnectionEdgeEnabled(spec)
}

func edgeTLSBaseDomain(spec obelisk.Spec, backhaul *peer.BackhaulInfo) string {
	if obelisk.VPSEdge(spec) {
		return obelisk.PublicEdgeBaseDomain()
	}
	if backhaul != nil && backhaul.GatewayEdgeDomain != "" {
		return backhaul.GatewayEdgeDomain
	}
	return obelisk.PublicEdgeBaseDomain()
}
