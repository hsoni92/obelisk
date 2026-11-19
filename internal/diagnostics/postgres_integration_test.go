//go:build integration

package diagnostics

import (
	"context"
	"os"
	"testing"
)

func TestProbePostgresVPSEdgeIntegration(t *testing.T) {
	url := os.Getenv("OBELISK_POSTGRES_EDGE_URL")
	if url == "" {
		t.Skip("set OBELISK_POSTGRES_EDGE_URL")
	}
	checks, err := probePostgres(context.Background(), url)
	for _, c := range checks {
		t.Logf("%s ok=%v detail=%s", c.Name, c.OK, c.Detail)
	}
	if err != nil {
		t.Fatal(err)
	}
}