package peer

import (
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// AppendFederatedConnections adds gateway-facing URL blocks for advertised peer instances.
func AppendFederatedConnections(spec obelisk.Spec, reg *Registry, base obelisk.Connections) obelisk.Connections {
	if !obelisk.IsGateway(spec) || reg == nil {
		return base
	}
	for _, ft := range reg.FederatedTargets(spec, 0) {
		if !ft.Inst.Enabled {
			continue
		}
		mini := obelisk.Spec{
			Deployment: spec.Deployment,
			Instances:  []obelisk.ServiceInstance{ft.Inst},
		}
		built := obelisk.BuildConnections(mini, nil)
		if len(built.Instances) == 0 {
			continue
		}
		ic := built.Instances[0]
		ic.Origin = "federated"
		ic.PeerID = ft.PeerID
		ic.Reachable = boolPtr(ft.Reachable)
		base.Instances = append(base.Instances, ic)
	}
	if len(base.Instances) > 0 && obelisk.ConnectionEdgeEnabled(spec) {
		base.Notes = base.Notes + " Federated rows use gateway hostnames; TLS is L4 passthrough to the peer (OQ1). Unreachable peers stay listed but are not proxied (OQ5)."
	}
	return base
}

func boolPtr(v bool) *bool { return &v }
