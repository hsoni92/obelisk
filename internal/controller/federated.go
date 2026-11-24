package controller

import (
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func (r *Reconciler) federatedTargets(spec obelisk.Spec) ([]peer.FederatedTarget, error) {
	fed, err := FederatedTargetsForConfig(r.ConfigPath, spec)
	if err != nil {
		return nil, err
	}
	return exposureFederated(spec, fed), nil
}
