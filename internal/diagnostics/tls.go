package diagnostics

import (
	"crypto/tls"
	"strings"
)

// EdgeTLSConfig is the TLS profile Obelisk edge clients typically use (SNI + auto-generated cert).
func EdgeTLSConfig(serverName string) *tls.Config {
	serverName = strings.TrimSpace(serverName)
	if serverName == "" {
		return &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}
	}
	return &tls.Config{
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}
}

// PostgresEdgeTLSConfig is EdgeTLSConfig with ALPN required for sslnegotiation=direct (PostgreSQL 17+).
func PostgresEdgeTLSConfig(serverName string) *tls.Config {
	cfg := EdgeTLSConfig(serverName)
	cfg.NextProtos = []string{"postgresql"}
	return cfg
}
