package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// DataDump is one uploadable backup artifact for an instance.
type DataDump struct {
	InstanceID   string
	ServiceType  string
	ObjectKey    string
	Body         []byte
	Skipped      bool
	SkipReason   string
}

// CollectDataDumps gathers logical backups for enabled running data instances.
func CollectDataDumps(ctx context.Context, spec obelisk.Spec, states map[string]string) []DataDump {
	var out []DataDump
	useK8s := K8sDataBackupEnabled()
	for _, inst := range spec.Instances {
		if !inst.Enabled || !obelisk.InstanceDataBackupEnabled(inst) {
			continue
		}
		if states != nil {
			st := states[inst.StateKey()]
			if st != "" && st != "running" {
				ts := time.Now().UTC().Format("20060102T150405Z")
				out = append(out, DataDump{
					InstanceID: inst.ID, ServiceType: inst.Type,
					ObjectKey:  obelisk.BackupDataObjectKey(inst.Type, inst.ID, ts+dataExt(inst.Type)),
					Skipped:    true,
					SkipReason: fmt.Sprintf("instance state is %q (start the service first)", st),
				})
				continue
			}
		}
		ts := time.Now().UTC().Format("20060102T150405Z")
		key := obelisk.BackupDataObjectKey(inst.Type, inst.ID, ts+dataExt(inst.Type))
		var body []byte
		var err error
		if useK8s {
			body, err = dumpInstanceK8s(ctx, inst)
		} else if inst.Type == "postgres" || inst.Type == "mysql" || inst.Type == "mongodb" {
			body, err = dumpInstanceLocal(ctx, inst)
		} else {
			out = append(out, DataDump{
				InstanceID: inst.ID, ServiceType: inst.Type, ObjectKey: key,
				Skipped:    true,
				SkipReason: "no cluster (run make dev-cluster or OBELISK_RECONCILE=cluster) and no local pg_dump/mysqldump/mongodump",
			})
			continue
		}
		if err != nil {
			out = append(out, DataDump{
				InstanceID: inst.ID, ServiceType: inst.Type, ObjectKey: key,
				Skipped: true, SkipReason: err.Error(),
			})
			continue
		}
		if len(body) == 0 {
			out = append(out, DataDump{
				InstanceID: inst.ID, ServiceType: inst.Type, ObjectKey: key,
				Skipped: true, SkipReason: "dump produced empty output",
			})
			continue
		}
		out = append(out, DataDump{
			InstanceID: inst.ID, ServiceType: inst.Type, ObjectKey: key, Body: body,
		})
	}
	return out
}

func countEligibleDataInstances(spec obelisk.Spec, states map[string]string) int {
	n := 0
	for _, inst := range spec.Instances {
		if !inst.Enabled || !obelisk.InstanceDataBackupEnabled(inst) {
			continue
		}
		if states != nil {
			st := states[inst.StateKey()]
			if st != "" && st != "running" {
				continue
			}
		}
		n++
	}
	return n
}

func dataExt(typ string) string {
	switch typ {
	case "mongodb":
		return ".archive"
	case "rustfs":
		return ".tar"
	case "qdrant":
		return ".snapshot"
	default:
		return ".sql"
	}
}

func dumpInstanceK8s(ctx context.Context, inst obelisk.ServiceInstance) ([]byte, error) {
	release := inst.ReleaseName()
	pod := release + "-0"
	ns := obeliskNamespace()
	switch inst.Type {
	case "postgres":
		inner := `export PGPASSWORD="${POSTGRES_PASSWORD}"; pg_dump -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" --no-owner --no-acl`
		return kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "postgres", "--", "sh", "-ec", inner)
	case "mysql":
		inner := `mysqldump -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}"`
		return kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "mysql", "--", "sh", "-ec", inner)
	case "mongodb":
		inner := `mongodump --archive --username="${MONGO_INITDB_ROOT_USERNAME}" --password="${MONGO_INITDB_ROOT_PASSWORD}" --authenticationDatabase=admin`
		return kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "mongodb", "--", "sh", "-ec", inner)
	case "rustfs", "qdrant":
		return nil, fmt.Errorf("%s data backup worker not implemented yet", inst.Type)
	default:
		return nil, fmt.Errorf("unsupported type %q", inst.Type)
	}
}

func strOr(s, def string) string {
	if strings.TrimSpace(s) != "" {
		return s
	}
	return def
}