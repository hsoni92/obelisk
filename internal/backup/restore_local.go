package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func restoreInstanceLocal(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	switch inst.Type {
	case "postgres":
		return restorePostgresLocal(ctx, inst, dump)
	case "mysql":
		return restoreMySQLLocal(ctx, inst, dump)
	case "mongodb":
		return restoreMongoLocal(ctx, inst, dump)
	default:
		return fmt.Errorf("no local restore for %q", inst.Type)
	}
}

func restorePostgresLocal(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	if _, err := exec.LookPath("psql"); err != nil {
		return fmt.Errorf("psql not on PATH")
	}
	host := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := envPort("OBELISK_POSTGRES_PORT", 5432)
	user, _, db := obelisk.ResolvedDBCreds(inst)
	tmp, err := os.CreateTemp("", "obelisk-restore-*.sql")
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(dump); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	run := func(u string) error {
		cmd := exec.CommandContext(ctx, "psql",
			"-h", host, "-p", port, "-U", u, "-d", db,
			"-v", "ON_ERROR_STOP=1",
			"-f", path,
		)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+inst.Credentials.Password)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if err := run(user); err != nil && user != "postgres" && strings.Contains(err.Error(), "does not exist") {
		return run("postgres")
	} else {
		return err
	}
}

func restoreMySQLLocal(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	if _, err := exec.LookPath("mysql"); err != nil {
		return fmt.Errorf("mysql client not on PATH")
	}
	host := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := envPort("OBELISK_MYSQL_PORT", 3306)
	user, _, db := obelisk.ResolvedDBCreds(inst)
	cmd := exec.CommandContext(ctx, "mysql",
		"-h", host, "-P", port, "-u", user, db,
	)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+inst.Credentials.Password)
	cmd.Stdin = strings.NewReader(string(dump))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func restoreMongoLocal(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	if _, err := exec.LookPath("mongorestore"); err != nil {
		return fmt.Errorf("mongorestore not on PATH")
	}
	host := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := envPort("OBELISK_MONGODB_PORT", 27017)
	user := strOr(inst.Credentials.Username, "obelisk")
	tmp, err := os.CreateTemp("", "obelisk-restore-*.archive")
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(dump); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "mongorestore",
		"--archive="+path,
		"--drop",
		"-h", host+":"+port,
		"-u", user,
		"-p", inst.Credentials.Password,
		"--authenticationDatabase", "admin",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}