package diagnostics

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func probePostgres(ctx context.Context, connURL string) ([]Check, error) {
	checks := []Check{{Name: "parse_url", OK: true}}
	u, err := url.Parse(connURL)
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, err
	}
	host := u.Hostname()
	if host == "" {
		return append(checks, Check{Name: "host", OK: false, Detail: "missing hostname"}),
			fmt.Errorf("postgres url missing hostname")
	}

	cfg, err := pgx.ParseConfig(connURL)
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, err
	}
	if edgeTLSFromURL(u, connURL) {
		cfg.TLSConfig = PostgresEdgeTLSConfig(host)
		cfg.SSLNegotiation = "direct"
		cfg.Fallbacks = nil
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		checks = append(checks, Check{Name: "connect", OK: false, Detail: err.Error()})
		return checks, err
	}
	defer conn.Close(ctx)

	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		checks = append(checks, Check{Name: "select_1", OK: false, Detail: err.Error()})
		return checks, err
	}
	checks = append(checks, Check{Name: "select_1", OK: true})
	return checks, nil
}

func edgeTLSFromURL(u *url.URL, connURL string) bool {
	port := u.Port()
	if port != "" && port != "5432" {
		return true
	}
	low := strings.ToLower(connURL)
	return strings.Contains(low, "sslmode=require") ||
		strings.Contains(low, "sslmode=verify") ||
		strings.Contains(low, "sslnegotiation=direct")
}