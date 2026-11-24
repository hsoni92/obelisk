package diagnostics

import "testing"

func TestPostgresEdgeTLSConfig_ALPN(t *testing.T) {
	cfg := PostgresEdgeTLSConfig("postgres.example.sslip.io")
	if cfg.ServerName != "postgres.example.sslip.io" {
		t.Fatalf("ServerName=%q", cfg.ServerName)
	}
	if len(cfg.NextProtos) != 1 || cfg.NextProtos[0] != "postgresql" {
		t.Fatalf("NextProtos=%v", cfg.NextProtos)
	}
}
