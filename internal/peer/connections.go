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
		ic := obelisk.InstanceConnection{
			ID:       ft.Inst.ID,
			Type:     ft.Inst.Type,
			Origin:   "federated",
			PeerID:   ft.PeerID,
			RouteID:  ft.Inst.ID,
			Reachable: boolPtr(ft.Reachable),
		}
		if h := obelisk.PublicEdgeHost(ft.Inst); h != "" {
			ic.PublicPath = h
		}
		mergeFederatedURLs(spec, ft.Inst, &ic)
		base.Instances = append(base.Instances, ic)
	}
	if len(base.Instances) > 0 && obelisk.ConnectionEdgeEnabled(spec) {
		base.Notes = base.Notes + " Federated rows use gateway hostnames; TLS is L4 passthrough to the peer (OQ1). Unreachable peers stay listed but are not proxied (OQ5)."
	}
	return base
}

func boolPtr(v bool) *bool { return &v }

func mergeFederatedURLs(spec obelisk.Spec, inst obelisk.ServiceInstance, ic *obelisk.InstanceConnection) {
	_ = spec
	host := obelisk.PublicEdgeHost(inst)
	if host == "" {
		return
	}
	switch inst.Type {
	case "postgres":
		ic.Postgres = "postgresql://USER:PASSWORD@" + host + ":" + obelisk.NativeEdgeTLSPort() + "/DATABASE"
	case "redis":
		ic.Redis = "rediss://:PASSWORD@" + host + ":" + obelisk.NativeEdgeTLSPort() + "/0"
	case "rustfs":
		ic.S3HTTP = obelisk.PublicHTTPEdgeURL(inst)
	}
}