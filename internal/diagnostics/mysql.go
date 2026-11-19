package diagnostics

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

// mysql://user:pass@tcp(host:port)/db?tls=true
var mysqlURLRe = regexp.MustCompile(`^mysql://([^:]+):([^@]+)@tcp\(([^:]+):(\d+)\)/([^?]+)`)

func probeMySQL(ctx context.Context, connURL string) ([]Check, error) {
	checks := []Check{{Name: "parse_url", OK: true}}
	m := mysqlURLRe.FindStringSubmatch(connURL)
	if m == nil {
		// local profile: mysql://user:pass@host:3306/db
		u, err := url.Parse(connURL)
		if err != nil {
			checks[0].OK = false
			checks[0].Detail = err.Error()
			return checks, err
		}
		user := u.User.Username()
		pass, _ := u.User.Password()
		host := u.Hostname()
		port := u.Port()
		if port == "" {
			port = "3306"
		}
		db := strings.TrimPrefix(u.Path, "/")
		return dialMySQL(ctx, checks, user, pass, host, port, db, false)
	}
	user, pass, host, port, db := m[1], m[2], m[3], m[4], m[5]
	user, _ = url.PathUnescape(user)
	pass, _ = url.PathUnescape(pass)
	tlsEdge := strings.Contains(connURL, "tls=true")
	return dialMySQL(ctx, checks, user, pass, host, port, db, tlsEdge)
}

func dialMySQL(ctx context.Context, checks []Check, user, pass, host, port, db string, tlsEdge bool) ([]Check, error) {
	if host == "" {
		checks = append(checks, Check{Name: "host", OK: false, Detail: "missing hostname"})
		return checks, fmt.Errorf("mysql url missing hostname")
	}
	tlsName := "obelisk-" + strings.ReplaceAll(strings.ReplaceAll(host, ".", "-"), ":", "-")
	if tlsEdge {
		_ = mysql.RegisterTLSConfig(tlsName, EdgeTLSConfig(host))
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=15s&readTimeout=10s", user, pass, host, port, db)
	if tlsEdge {
		dsn += "&tls=" + tlsName
	} else {
		dsn += "&tls=false"
	}

	dbh, err := sql.Open("mysql", dsn)
	if err != nil {
		checks = append(checks, Check{Name: "connect", OK: false, Detail: err.Error()})
		return checks, err
	}
	defer dbh.Close()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := dbh.PingContext(ctx); err != nil {
		checks = append(checks, Check{Name: "ping", OK: false, Detail: err.Error()})
		return checks, err
	}
	var one int
	if err := dbh.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		checks = append(checks, Check{Name: "select_1", OK: false, Detail: err.Error()})
		return checks, err
	}
	checks = append(checks, Check{Name: "select_1", OK: true})
	return checks, nil
}