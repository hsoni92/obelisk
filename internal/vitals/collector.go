package vitals

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Platform workload keys in the vitals panel.
const (
	KeyController = "platform/controller"
	KeyAdminUI    = "platform/admin-ui"
	KeyKong       = "platform/kong"
)

type Collector struct {
	Namespace string
}

func (c Collector) namespace() string {
	if c.Namespace != "" {
		return c.Namespace
	}
	return "obelisk"
}

type collectInput struct {
	Spec     obelisk.Spec
	States   map[string]string
	Meta     rowMeta
	DoDisk   bool
}

// CollectMinute samples memory and CPU; disk may be carried or refreshed.
func (c Collector) CollectMinute(ctx context.Context, store *Store, in collectInput) {
	if controller.ReconcileDisabled() {
		return
	}
	ns := c.namespace()
	now := time.Now().UTC()

	caps, capErr := kubectlNodeCapacity(ctx)
	nodeTop, topErr := kubectlTopNode(ctx)
	podTops, podsErr := kubectlTopPods(ctx, ns)

	if podsErr != nil {
		return
	}
	if capErr != nil {
		caps = nodeCaps{}
	}

	machineMemTotal := caps.MemoryBytes
	machineMemUsed := nodeTop.MemoryBytes
	if topErr != nil {
		var sum int64
		for _, p := range podTops {
			sum += p.MemBytes
		}
		machineMemUsed = sum
	}
	machineCPUPercent := 0.0
	if caps.CPUMilli > 0 {
		cpuUsed := nodeTop.CPUMilli
		if topErr != nil {
			var sum int64
			for _, p := range podTops {
				sum += p.CPUMilli
			}
			cpuUsed = sum
		}
		machineCPUPercent = 100 * float64(cpuUsed) / float64(caps.CPUMilli)
	}
	machineDiskTotal := caps.DiskBytes
	var machineDiskUsed int64
	hostDiskOK := false
	if u, t, ok := hostRootDisk(); ok {
		machineDiskUsed = u
		hostDiskOK = true
		if machineDiskTotal == 0 || t < machineDiskTotal {
			machineDiskTotal = t
		}
	}

	if in.DoDisk {
		if job, ok := store.NextDiskJob(); ok {
			if u, t, labels, obs, ok := c.measureDisk(ctx, ns, job); ok {
				store.SetDiskHold(job.RowKey, diskHold{
					Used: u, Total: t, Known: true, Observed: obs, Labels: labels,
				})
			}
		}
	}

	var accountedDisk int64

	for _, def := range in.Meta.keys {
		mem, cpu, pi := c.metricsForKey(ctx, ns, def, podTops)
		diskUsed, diskTotal, diskKnown, diskObs := c.diskForKey(store, def)

		if diskKnown {
			accountedDisk += diskUsed
		}

		memTotal := machineMemTotal
		memKnown := mem >= 0
		memUsed := mem
		cpuKnown := cpu >= 0
		cpuShare := 0.0
		if cpuKnown && caps.CPUMilli > 0 {
			cpuShare = 100 * float64(cpu) / float64(caps.CPUMilli)
		}

		slot := Slot{
			At:                now,
			MemoryUsed:        memUsed,
			MemoryTotal:       memTotal,
			MemoryKnown:       memKnown && (def.Running || memUsed > 0),
			CPUSharePercent:   cpuShare,
			CPUKnown:          cpuKnown && def.Running,
			DiskUsed:          diskUsed,
			DiskTotal:         diskTotal,
			DiskKnown:         diskKnown,
			DiskObserved:      diskObs,
			RestartCount:      pi.RestartCount,
			LastTerminationReason: pi.LastReason,
		}
		if !def.Running {
			slot.MemoryKnown = false
			slot.CPUKnown = false
		}
		store.AppendRow(def.Key, slot)
	}

	if !hostDiskOK {
		machineDiskUsed = accountedDisk
	}
	machineSlot := Slot{
		At:              now,
		MemoryUsed:      machineMemUsed,
		MemoryTotal:     machineMemTotal,
		MemoryKnown:     machineMemTotal > 0,
		CPUSharePercent: machineCPUPercent,
		CPUKnown:        caps.CPUMilli > 0,
		DiskUsed:        machineDiskUsed,
		DiskTotal:       machineDiskTotal,
		DiskKnown:       machineDiskTotal > 0,
		DiskObserved:    now,
	}
	store.AppendMachine(machineSlot)
}

func (c Collector) metricsForKey(ctx context.Context, ns string, def rowDef, tops map[string]podTop) (memBytes, cpuMilli int64, pi podInspect) {
	if !def.Running {
		return 0, 0, podInspect{}
	}
	pods, err := c.podsForKey(ctx, ns, def.Key)
	if err != nil || len(pods) == 0 {
		return -1, -1, podInspect{}
	}
	pod := pods[0]
	if t, ok := tops[pod]; ok {
		memBytes = t.MemBytes
		cpuMilli = t.CPUMilli
	} else {
		memBytes = -1
		cpuMilli = -1
	}
	pi, _ = kubectlPodInspect(ctx, ns, pod)
	return memBytes, cpuMilli, pi
}

func (c Collector) podsForKey(ctx context.Context, ns, key string) ([]string, error) {
	switch key {
	case KeyController:
		return listPodsForApp(ctx, ns, "app.kubernetes.io/name=obelisk-controller")
	case KeyAdminUI:
		return listPodsForApp(ctx, ns, "app=obelisk-admin-ui")
	case KeyKong:
		return listPodsForApp(ctx, ns, "app=obelisk-kong")
	default:
		// instance type/id -> release name
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad key %q", key)
		}
		release := obelisk.ServiceInstance{Type: parts[0], ID: parts[1]}.ReleaseName()
		return listPodsForApp(ctx, ns, "app="+release)
	}
}

func (c Collector) diskForKey(store *Store, def rowDef) (used, total int64, known bool, observed time.Time) {
	if h, ok := store.DiskHold(def.Key); ok && h.Known {
		return h.Used, h.Total, true, h.Observed
	}
	return 0, 0, false, time.Time{}
}

func (c Collector) measureDisk(ctx context.Context, ns string, job diskJob) (used, total int64, labels []string, observed time.Time, ok bool) {
	pods, err := listPodsForApp(ctx, ns, "app="+job.Release)
	if err != nil || len(pods) == 0 {
		return 0, 0, nil, time.Time{}, false
	}
	pod := pods[0]
	var sumUsed int64
	var maxTotal int64
	for _, m := range job.Mounts {
		u, t, err := kubectlDfMount(ctx, ns, pod, m.Container, m.Path)
		if err != nil {
			continue
		}
		sumUsed += u
		if t > maxTotal {
			maxTotal = t
		}
		labels = append(labels, m.Label)
	}
	if maxTotal == 0 {
		return 0, 0, labels, time.Time{}, false
	}
	return sumUsed, maxTotal, labels, time.Now().UTC(), true
}

func diskJobsForSpec(spec obelisk.Spec) []diskJob {
	var jobs []diskJob
	for _, inst := range spec.Instances {
		key := inst.StateKey()
		mounts := dataMounts(inst.Type)
		if len(mounts) == 0 {
			continue
		}
		jobs = append(jobs, diskJob{
			RowKey:  key,
			Release: inst.ReleaseName(),
			Mounts:  mounts,
		})
	}
	return jobs
}

func dataMounts(typ string) []diskMount {
	switch typ {
	case "postgres":
		return []diskMount{{Container: "postgres", Path: "/var/lib/postgresql/data", Label: "data"}}
	case "mysql":
		return []diskMount{{Container: "mysql", Path: "/var/lib/mysql", Label: "data"}}
	case "mongodb":
		return []diskMount{{Container: "mongodb", Path: "/data/db", Label: "data"}}
	case "redis":
		return []diskMount{{Container: "redis", Path: "/data", Label: "data"}}
	case "rustfs":
		return []diskMount{{Container: "rustfs", Path: "/data", Label: "data"}}
	case "qdrant":
		return []diskMount{{Container: "qdrant", Path: "/qdrant/storage", Label: "data"}}
	case "searxng":
		return []diskMount{
			{Container: "searxng", Path: "/etc/searxng", Label: "config"},
			{Container: "searxng", Path: "/var/cache/searxng", Label: "data"},
		}
	default:
		return nil
	}
}

// BuildMeta lists vitals rows for the current spec and observed states.
func BuildMeta(spec obelisk.Spec, states map[string]string) rowMeta {
	meta := rowMeta{
		keys: []rowDef{
			{Key: KeyController, Name: "Controller", Kind: "platform", Running: true},
			{Key: KeyAdminUI, Name: "Admin UI", Kind: "platform", Running: true},
			{Key: KeyKong, Name: "Kong", Kind: "platform", Running: true},
		},
	}
	for _, inst := range spec.Instances {
		st := states[inst.StateKey()]
		running := inst.Enabled && st == "running"
		name := inst.DisplayName
		if name == "" {
			name = inst.ID
		}
		meta.keys = append(meta.keys, rowDef{
			Key:     inst.StateKey(),
			Name:    name,
			Kind:    "instance",
			Type:    inst.Type,
			Running: running,
		})
	}
	return meta
}