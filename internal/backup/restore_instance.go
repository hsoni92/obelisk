package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// RestorePostgresSQL applies a plain SQL dump to a running postgres instance.
func RestorePostgresSQL(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	if inst.Type != "postgres" {
		return fmt.Errorf("restore not supported for %q", inst.Type)
	}
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
	cmd := exec.CommandContext(ctx, "psql",
		"-h", host, "-p", port, "-U", user, "-d", db,
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