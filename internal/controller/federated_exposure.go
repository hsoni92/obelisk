package controller

import (
	"github.com/hsoni92/obelisk/internal/peer"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// exposureFederated returns only routes that should be programmed on the gateway edge (OQ4, OQ5).
func exposureFederated(spec obelisk.Spec, fed []peer.FederatedTarget) []peer.FederatedTarget {
	exposeNative := obelisk.PeerGatewayExposeNativePublic(spec)
	out := make([]peer.FederatedTarget, 0, len(fed))
	for _, ft := range fed {
		if !ft.Reachable {
			continue
		}
		if ft.NativePort > 0 && !exposeNative {
			ft.NativePort = 0
		}
		if ft.NativePort <= 0 && ft.HTTPPort <= 0 {
			continue
		}
		out = append(out, ft)
	}
	return out
}