package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func dumpInstanceLocal(ctx context.Context, inst obelisk.ServiceInstance) ([]byte, error) {
	host := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	switch inst.Type {
	case "postgres":
		if _, err := exec.LookPath("pg_dump"); err != nil {
			return nil, fmt.Errorf("pg_dump not on PATH (port-forward Postgres or use make dev-cluster)")
		}
		port := envPort("OBELISK_POSTGRES_PORT", 5432)
		user := strOr(inst.Credentials.Username, "obelisk")
		db := strOr(inst.Credentials.Database, "obelisk")
		cmd := exec.CommandContext(ctx, "pg_dump",
			"-h", host, "-p", port, "-U", user, "-d", db, "--no-owner", "--no-acl",
		)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+inst.Credentials.Password)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return out, nil
	case "mysql":
		if _, err := exec.LookPath("mysqldump"); err != nil {
			return nil, fmt.Errorf("mysqldump not on PATH")
		}
		port := envPort("OBELISK_MYSQL_PORT", 3306)
		user := strOr(inst.Credentials.Username, "obelisk")
		db := strOr(inst.Credentials.Database, "obelisk")
		cmd := exec.CommandContext(ctx, "mysqldump",
			"-h", host, "-P", port, "-u", user, db,
		)
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+inst.Credentials.Password)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return out, nil
	case "mongodb":
		if _, err := exec.LookPath("mongodump"); err != nil {
			return nil, fmt.Errorf("mongodump not on PATH (port-forward MongoDB or use make dev-cluster)")
		}
		port := envPort("OBELISK_MONGODB_PORT", 27017)
		user := strOr(inst.Credentials.Username, "obelisk")
		cmd := exec.CommandContext(ctx, "mongodump",
			"--archive",
			"-h", host+":"+port,
			"-u", user,
			"-p", inst.Credentials.Password,
			"--authenticationDatabase", "admin",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("no local dump for %q", inst.Type)
	}
}

func envPort(key string, def int) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return strconv.Itoa(def)
}