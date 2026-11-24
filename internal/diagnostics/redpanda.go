package diagnostics

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

func probeRedpanda(ctx context.Context, bootstrap string) ([]Check, error) {
	checks := []Check{{Name: "parse_bootstrap", OK: true}}
	host, port, err := net.SplitHostPort(strings.TrimSpace(bootstrap))
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, fmt.Errorf("parse kafka bootstrap: %w", err)
	}
	if host == "" || port == "" {
		checks[0].OK = false
		checks[0].Detail = "missing host or port"
		return checks, fmt.Errorf("kafka bootstrap missing host or port")
	}

	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		checks = append(checks, Check{Name: "tcp_connect", OK: false, Detail: err.Error()})
		return checks, err
	}
	_ = conn.Close()
	checks = append(checks, Check{Name: "tcp_connect", OK: true})
	return checks, nil
}
