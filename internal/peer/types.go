package peer

import (
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// AdvertiseRequest is sent by a local appliance to the gateway.
type AdvertiseRequest struct {
	PeerID             string               `json:"peer_id"`
	MachineID          string               `json:"machine_id"`
	TailnetHost        string               `json:"tailnet_host"`
	DeploymentRole     string               `json:"deployment_role,omitempty"`
	NativeBackhaulPort int                  `json:"native_backhaul_port,omitempty"`
	HTTPBackhaulPort   int                  `json:"http_backhaul_port,omitempty"`
	Instances          []AdvertisedInstance `json:"instances"`
}

type AdvertisedInstance struct {
	Type        string                      `json:"type"`
	ID          string                      `json:"id"`
	Enabled     bool                        `json:"enabled"`
	DisplayName string                      `json:"display_name,omitempty"`
	Credentials obelisk.InstanceCredentials `json:"credentials,omitempty"`
	Databases   []string                    `json:"databases,omitempty"`
}

// AdvertiseResponse may return a newly issued peer token.
type AdvertiseResponse struct {
	Accepted       bool   `json:"accepted"`
	Token          string `json:"token,omitempty"`
	Message        string `json:"message,omitempty"`
	EdgeBaseDomain string `json:"edge_base_domain,omitempty"`
}

// FederatedTarget is an exposure upstream for a peer-owned instance.
type FederatedTarget struct {
	PeerID       string
	Reachable    bool
	Inst         obelisk.ServiceInstance
	UpstreamHost string
	NativePort   int
	HTTPPort     int
}

// PeerRecord is runtime state for one local peer on the gateway.
type PeerRecord struct {
	ID                 string                        `json:"id"`
	MachineID          string                        `json:"machine_id,omitempty"`
	TailnetHost        string                        `json:"tailnet_host,omitempty"`
	NativeBackhaulPort int                           `json:"native_backhaul_port,omitempty"`
	HTTPBackhaulPort   int                           `json:"http_backhaul_port,omitempty"`
	Admitted           bool                          `json:"admitted"`
	TokenHash          string                        `json:"token_hash,omitempty"`
	LastSeen           time.Time                     `json:"last_seen"`
	Reachable          bool                          `json:"reachable"`
	Instances          map[string]AdvertisedInstance `json:"instances,omitempty"`
}
