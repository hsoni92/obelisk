package controller

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func (r *Reconciler) ensureInstanceDatabases(ctx context.Context, inst obelisk.ServiceInstance) error {
	if r.reconcileDisabled() || !inst.Enabled || !obelisk.IsRelationalEngine(inst.Type) {
		return nil
	}
	pod, err := r.readyPodName(ctx, inst.ReleaseName())
	if err != nil {
		return err
	}
	user, pass, _ := obelisk.ResolvedDBCreds(inst)
	if pass == "" {
		return fmt.Errorf("missing database password for %s", inst.StateKey())
	}
	names := obelisk.DesiredDatabaseNames(inst)
	switch inst.Type {
	case "postgres":
		return ensurePostgresDatabases(ctx, r.namespace(), pod, user, names)
	case "mysql":
		return ensureMySQLDatabases(ctx, r.namespace(), pod, user, pass, names)
	case "mongodb":
		return ensureMongoDatabases(ctx, r.namespace(), pod, user, pass, names)
	default:
		return nil
	}
}

func (r *Reconciler) readyPodName(ctx context.Context, release string) (string, error) {
	pods := r.PodsForRelease(ctx, release)
	if pods.Err != nil {
		return "", pods.Err
	}
	if pods.Ready < 1 || pods.Total < 1 {
		return "", fmt.Errorf("no ready pod for %s", release)
	}
	ns := r.namespace()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", ns,
		"-l", fmt.Sprintf("app=%s", release),
		"-o", "jsonpath={.items[?(@.status.phase==\"Running\")].metadata.name}")
	cmd.Env = kubeconfigEnv()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("list pods: %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", fmt.Errorf("no running pod for %s", release)
	}
	// jsonpath may return space-separated; take first.
	if i := strings.IndexByte(name, ' '); i > 0 {
		name = name[:i]
	}
	return name, nil
}

func kubectlExec(ctx context.Context, ns, pod, container string, argv ...string) error {
	args := []string{"exec", "-n", ns, pod, "-c", container, "--"}
	args = append(args, argv...)
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = kubeconfigEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func ensurePostgresDatabases(ctx context.Context, ns, pod, user string, names []string) error {
	for _, db := range names {
		create := fmt.Sprintf("CREATE DATABASE %s OWNER %s", pgQuoteIdent(db), pgQuoteIdent(user))
		if err := kubectlExec(ctx, ns, pod, "postgres", "psql", "-U", user, "-d", "postgres", "-c", create); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "already exists") || strings.Contains(msg, "42P04") {
				continue
			}
			return err
		}
	}
	return nil
}

func ensureMySQLDatabases(ctx context.Context, ns, pod, user, pass string, names []string) error {
	for _, db := range names {
		sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", mysqlQuoteIdent(db))
		if err := kubectlExec(ctx, ns, pod, "mysql", "mysql", "-uroot", "-p"+pass, "-e", sql); err != nil {
			return err
		}
		if user == "root" {
			continue
		}
		grant := fmt.Sprintf("GRANT ALL ON `%s`.* TO '%s'@'%%'", mysqlQuoteIdent(db), mysqlQuoteIdent(user))
		if err := kubectlExec(ctx, ns, pod, "mysql", "mysql", "-uroot", "-p"+pass, "-e", grant); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "already exists") {
				continue
			}
			return err
		}
	}
	return nil
}

func ensureMongoDatabases(ctx context.Context, ns, pod, user, pass string, names []string) error {
	for _, db := range names {
		script := fmt.Sprintf(
			`db.getSiblingDB(%s).createCollection('_obelisk_init');`,
			mongoQuoteLiteral(db),
		)
		auth := ""
		if user != "" && pass != "" {
			auth = fmt.Sprintf("-u %s -p %s --authenticationDatabase admin ", mongoQuoteShell(user), mongoQuoteShell(pass))
		}
		cmd := fmt.Sprintf(`mongosh --quiet %s--eval %q`, auth, script)
		if err := kubectlExec(ctx, ns, pod, "mongodb", "sh", "-ec", cmd); err != nil {
			return err
		}
	}
	return nil
}

func pgQuoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func pgQuoteLiteral(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

func mysqlQuoteIdent(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

func mongoQuoteLiteral(s string) string {
	return pgQuoteLiteral(s)
}

func mongoQuoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}