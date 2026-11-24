package peer

import (
	"path/filepath"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestAppendFederatedConnections_PostgresURLs(t *testing.T) {
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	dir := t.TempDir()
	cfg := filepath.Join(dir, "obelisk.yaml")
	reg, err := LoadRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := reg.ApplyAdvertisement(AdvertiseRequest{
		PeerID:      "obelisk-470f4a4845ec",
		TailnetHost: "100.64.0.2",
		Instances: []AdvertisedInstance{{
			Type:    "postgres",
			ID:      "eager-stork",
			Enabled: true,
			Credentials: obelisk.InstanceCredentials{
				Username: "appuser",
				Password: "p@ss&word",
			},
		}},
	}, "", true)
	if err != nil || !resp.Accepted {
		t.Fatalf("admit: %+v err=%v", resp, err)
	}
	spec := obelisk.DefaultSpec("vps")
	spec.Deployment.Role = obelisk.DeploymentRoleGateway
	conns := AppendFederatedConnections(spec, reg, obelisk.Connections{})
	if len(conns.Instances) != 1 {
		t.Fatalf("instances %+v", conns.Instances)
	}
	ic := conns.Instances[0]
	if ic.Origin != "federated" || ic.PeerID != "obelisk-470f4a4845ec" {
		t.Fatalf("metadata %+v", ic)
	}
	want := "postgresql://appuser:p%40ss%26word@postgres-obelisk-470f4a4845ec-eager-stork.178-105-35-230.sslip.io:9443/default?" + obelisk.VPSPostgresURLQuery
	if ic.Postgres != want {
		t.Fatalf("postgres %q want %q", ic.Postgres, want)
	}
}
