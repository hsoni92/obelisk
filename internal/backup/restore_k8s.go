package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func restoreInstanceK8s(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	release := inst.ReleaseName()
	pod := release + "-0"
	ns := obeliskNamespace()
	user, pass, db := obelisk.ResolvedDBCreds(inst)
	switch inst.Type {
	case "postgres":
		inner := fmt.Sprintf(`PGPASSWORD=%q psql -U %q -d %q -v ON_ERROR_STOP=1 -f -`, pass, user, db)
		if err := kubectlStdin(ctx, bytes.NewReader(dump), "exec", "-i", "-n", ns, pod, "-c", "postgres", "--", "sh", "-ec", inner); err != nil {
			if user != "postgres" && strings.Contains(err.Error(), "does not exist") {
				inner = fmt.Sprintf(`PGPASSWORD=%q psql -U postgres -d %q -v ON_ERROR_STOP=1 -f -`, pass, db)
				return kubectlStdin(ctx, bytes.NewReader(dump), "exec", "-i", "-n", ns, pod, "-c", "postgres", "--", "sh", "-ec", inner)
			}
			return err
		}
		return nil
	case "mysql":
		inner := fmt.Sprintf(`mysql -u%q -p%q %q`, user, pass, db)
		return kubectlStdin(ctx, bytes.NewReader(dump), "exec", "-i", "-n", ns, pod, "-c", "mysql", "--", "sh", "-ec", inner)
	case "mongodb":
		inner := fmt.Sprintf(
			`mongorestore --archive --drop --username=%q --password=%q --authenticationDatabase=admin`,
			user, pass,
		)
		return kubectlStdin(ctx, bytes.NewReader(dump), "exec", "-i", "-n", ns, pod, "-c", "mongodb", "--", "sh", "-ec", inner)
	case "rustfs":
		inner := fmt.Sprintf(`tar -xf - -C %s`, rustfsDataDir)
		return kubectlStdin(ctx, bytes.NewReader(dump), "exec", "-i", "-n", ns, pod, "-c", "rustfs", "--", "sh", "-ec", inner)
	case "qdrant":
		key := qdrantAPIKey(inst)
		if key == "" {
			return fmt.Errorf("qdrant api key missing in instance credentials")
		}
		return qdrantRestoreFullSnapshotK8s(ctx, ns, pod, key, dump)
	default:
		return fmt.Errorf("unsupported type %q", inst.Type)
	}
}

func kubectlStdin(ctx context.Context, stdin io.Reader, args ...string) error {
	_, err := kubectlCombined(ctx, stdin, args...)
	return err
}