package controller

import (
	"context"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/exposure"
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// ReconcileExposurePeers reloads federated routes on the gateway after peer catalog changes.
func (r *Reconciler) ReconcileExposurePeers(ctx context.Context, configPath string) error {
	cfg := config.EnsureReadablePath(configPath)
	spec, err := config.Load(cfg)
	if err != nil {
		return err
	}
	if !obelisk.IsGateway(spec) {
		return nil
	}
	reg, err := peer.LoadRegistry(cfg)
	if err != nil {
		return err
	}
	fed := exposureFederated(spec, reg.FederatedTargets(spec, 0))
	return exposure.Reconcile(ctx, spec, r.HelmRoot, r.Namespace, fed, configPath)
}

// FederatedTargetsForConfig loads peer registry targets for exposure reconcile.
func FederatedTargetsForConfig(configPath string, spec obelisk.Spec) ([]peer.FederatedTarget, error) {
	if !obelisk.IsGateway(spec) {
		return nil, nil
	}
	reg, err := peer.LoadRegistry(configPath)
	if err != nil {
		return nil, err
	}
	const stale = 2 * time.Minute
	reg.MarkStale(stale)
	return reg.FederatedTargets(spec, stale), nil
}
