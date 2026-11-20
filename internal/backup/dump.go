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
			body, err = dumpInstanceK8s(ctx, spec, inst)
		} else if inst.Type == "postgres" || inst.Type == "mysql" || inst.Type == "mongodb" {
			body, err = dumpInstanceLocal(ctx, inst)
		} else {
			out = append(out, DataDump{
				InstanceID: inst.ID, ServiceType: inst.Type, ObjectKey: key,
				Skipped:    true,
				SkipReason: "no cluster (run make dev or OBELISK_RECONCILE=cluster) and no local pg_dump/mysqldump/mongodump",
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

// CollectDataDumpsForInstance is like CollectDataDumps but scoped to one instance.
func CollectDataDumpsForInstance(ctx context.Context, spec obelisk.Spec, states map[string]string, serviceType, instanceID string) []DataDump {
	var out []DataDump
	for _, d := range CollectDataDumps(ctx, spec, states) {
		if d.ServiceType == serviceType && d.InstanceID == instanceID {
			out = append(out, d)
		}
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

func dumpInstanceK8s(ctx context.Context, spec obelisk.Spec, inst obelisk.ServiceInstance) ([]byte, error) {
	release := inst.ReleaseName()
	pod := release + "-0"
	ns := obeliskNamespace()
	user, pass, db := obelisk.ResolvedDBCreds(inst)
	mongoEdgeTLS := MongoEdgeTLSEnabled(ctx, spec)
	switch inst.Type {
	case "postgres":
		return postgresDumpK8s(ctx, ns, pod, user, pass, db)
	case "mysql":
		inner := fmt.Sprintf(`mysqldump -u%q -p%q %q`, user, pass, db)
		return kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "mysql", "--", "sh", "-ec", inner)
	case "mongodb":
		inner := fmt.Sprintf(
			`mongodump --archive --username=%q --password=%q --authenticationDatabase=admin%s`,
			user, pass, mongoDumpK8sTLSFlags(mongoEdgeTLS),
		)
		return kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "mongodb", "--", "sh", "-ec", inner)
	case "rustfs":
		return rustfsDumpK8s(ctx, ns, pod)
	case "qdrant":
		return qdrantDumpK8s(ctx, ns, pod, qdrantAPIKey(inst))
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

// postgresDumpK8s uses desired-state credentials, then retries as role postgres when
// the data volume was initialized before username/database were set on the instance.
func postgresDumpK8s(ctx context.Context, ns, pod, user, pass, db string) ([]byte, error) {
	inner := fmt.Sprintf(`PGPASSWORD=%q pg_dump -U %q -d %q --no-owner --no-acl`, pass, user, db)
	out, err := kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "postgres", "--", "sh", "-ec", inner)
	if err == nil {
		return out, nil
	}
	if user != "postgres" && strings.Contains(err.Error(), "does not exist") {
		inner = fmt.Sprintf(`PGPASSWORD=%q pg_dump -U postgres -d %q --no-owner --no-acl`, pass, db)
		return kubectlOutput(ctx, "exec", "-n", ns, pod, "-c", "postgres", "--", "sh", "-ec", inner)
	}
	return nil, err
}