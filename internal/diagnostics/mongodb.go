package diagnostics

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func probeMongoDB(ctx context.Context, connURL string) ([]Check, error) {
	checks := []Check{{Name: "parse_url", OK: true}}
	u, err := url.Parse(connURL)
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, err
	}
	host := u.Hostname()
	if host == "" {
		checks = append(checks, Check{Name: "host", OK: false, Detail: "missing hostname"})
		return checks, fmt.Errorf("mongodb url missing hostname")
	}

	opts := options.Client().ApplyURI(connURL).
		SetServerSelectionTimeout(15 * time.Second).
		SetConnectTimeout(15 * time.Second)
	if u.Query().Get("tls") == "true" || u.Port() != "27017" {
		opts.SetTLSConfig(EdgeTLSConfig(host))
	}

	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		checks = append(checks, Check{Name: "connect", OK: false, Detail: err.Error()})
		return checks, err
	}
	defer func() { _ = client.Disconnect(ctx) }()

	if err := client.Ping(ctx, nil); err != nil {
		checks = append(checks, Check{Name: "ping", OK: false, Detail: err.Error()})
		return checks, err
	}
	checks = append(checks, Check{Name: "ping", OK: true})
	return checks, nil
}