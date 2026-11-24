package diagnostics

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func probeRedis(ctx context.Context, connURL string) ([]Check, error) {
	checks := []Check{{Name: "parse_url", OK: true}}
	u, err := url.Parse(connURL)
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, fmt.Errorf("parse redis url: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		checks = append(checks, Check{Name: "host", OK: false, Detail: "missing hostname"})
		return checks, fmt.Errorf("redis url missing hostname")
	}

	opt, err := redis.ParseURL(connURL)
	if err != nil {
		checks = append(checks, Check{Name: "parse_url", OK: false, Detail: err.Error()})
		return checks, err
	}
	if strings.EqualFold(u.Scheme, "rediss") {
		if opt.TLSConfig == nil {
			opt.TLSConfig = EdgeTLSConfig(host)
		} else {
			opt.TLSConfig.ServerName = host
			opt.TLSConfig.InsecureSkipVerify = true
		}
	}
	opt.DialTimeout = 15 * time.Second
	opt.ReadTimeout = 10 * time.Second

	client := redis.NewClient(opt)
	defer func() { _ = client.Close() }()

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		checks = append(checks, Check{Name: "ping", OK: false, Detail: err.Error()})
		return checks, err
	}
	checks = append(checks, Check{Name: "ping", OK: true, Detail: pong})
	return checks, nil
}
