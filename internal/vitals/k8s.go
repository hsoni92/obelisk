package vitals

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/controller"
)

type podTop struct {
	Name     string
	CPUMilli int64
	MemBytes int64
}

type nodeCaps struct {
	MemoryBytes int64
	CPUMilli    int64
	DiskBytes   int64
}

type nodeTop struct {
	CPUMilli    int64
	MemoryBytes int64
}

func kubectlTopPods(ctx context.Context, ns string) (map[string]podTop, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "top", "pod", "-n", ns, "--no-headers")
	cmd.Env = controller.KubeconfigEnv()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	m := make(map[string]podTop)
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		cpu, ok1 := parseCPUQuantity(fields[1])
		mem, ok2 := parseBytesQuantity(fields[2])
		if !ok1 || !ok2 {
			continue
		}
		m[name] = podTop{Name: name, CPUMilli: cpu, MemBytes: mem}
	}
	return m, sc.Err()
}

func kubectlTopNode(ctx context.Context) (nodeTop, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "top", "node", "--no-headers")
	cmd.Env = controller.KubeconfigEnv()
	out, err := cmd.Output()
	if err != nil {
		return nodeTop{}, err
	}
	line := strings.TrimSpace(string(out))
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nodeTop{}, fmt.Errorf("unexpected top node: %q", line)
	}
	cpu, ok1 := parseCPUQuantity(fields[1])
	if !ok1 {
		return nodeTop{}, fmt.Errorf("parse top node cpu: %q", line)
	}
	// metrics-server prints either NAME CPU MEMORY or NAME CPU CPU% MEMORY MEMORY%.
	var mem int64
	var ok2 bool
	if len(fields) >= 5 {
		mem, ok2 = parseBytesQuantity(fields[3])
	} else if len(fields) >= 3 {
		mem, ok2 = parseBytesQuantity(fields[2])
	}
	if !ok2 {
		for i := len(fields) - 1; i >= 1; i-- {
			if b, ok := parseBytesQuantity(fields[i]); ok {
				mem = b
				ok2 = true
				break
			}
		}
	}
	if !ok2 {
		return nodeTop{}, fmt.Errorf("parse top node memory: %q", line)
	}
	return nodeTop{CPUMilli: cpu, MemoryBytes: mem}, nil
}

func kubectlNodeCapacity(ctx context.Context) (nodeCaps, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "-o", "json")
	cmd.Env = controller.KubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
		return nodeCaps{}, err
	}
	var doc struct {
		Items []struct {
			Status struct {
				Capacity   map[string]string `json:"capacity"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nodeCaps{}, err
	}
	if len(doc.Items) == 0 {
		return nodeCaps{}, fmt.Errorf("no nodes")
	}
	cap := doc.Items[0].Status.Capacity
	mem, _ := parseBytesQuantity(cap["memory"])
	cpu, _ := parseCPUQuantity(cap["cpu"])
	disk, _ := parseBytesQuantity(cap["ephemeral-storage"])
	return nodeCaps{MemoryBytes: mem, CPUMilli: cpu, DiskBytes: disk}, nil
}

type podInspect struct {
	RestartCount  int
	LastReason    string
	Phase         string
	RunningSince  time.Time
	Ready         bool
	WaitingReason string
}

func kubectlPodInspect(ctx context.Context, ns, pod string) (podInspect, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pod", pod, "-n", ns, "-o", "json")
	cmd.Env = controller.KubeconfigEnv()
	raw, err := cmd.Output()
	if err != nil {
		return podInspect{}, err
	}
	var doc struct {
		Status struct {
			Phase      string `json:"phase"`
			StartTime  string `json:"startTime"`
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
			ContainerStatuses []struct {
				RestartCount int  `json:"restartCount"`
				Ready        bool `json:"ready"`
				State        struct {
					Waiting struct {
						Reason string `json:"reason"`
					} `json:"waiting"`
					Running struct {
						StartedAt string `json:"startedAt"`
					} `json:"running"`
				} `json:"state"`
				LastState struct {
					Terminated struct {
						Reason string `json:"reason"`
					} `json:"terminated"`
				} `json:"lastState"`
			} `json:"containerStatuses"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return podInspect{}, err
	}
	pi := podInspect{Phase: doc.Status.Phase}
	for _, cond := range doc.Status.Conditions {
		if cond.Type == "Ready" && cond.Status == "True" {
			pi.Ready = true
			break
		}
	}
	if t, err := time.Parse(time.RFC3339, doc.Status.StartTime); err == nil {
		pi.RunningSince = t
	}
	var earliestRunning time.Time
	for _, c := range doc.Status.ContainerStatuses {
		pi.RestartCount += c.RestartCount
		if r := c.LastState.Terminated.Reason; r != "" {
			pi.LastReason = r
		}
		if !c.Ready && pi.WaitingReason == "" && c.State.Waiting.Reason != "" {
			pi.WaitingReason = c.State.Waiting.Reason
		}
		if started := c.State.Running.StartedAt; started != "" {
			if t, err := time.Parse(time.RFC3339, started); err == nil {
				if earliestRunning.IsZero() || t.Before(earliestRunning) {
					earliestRunning = t
				}
			}
		}
	}
	if !earliestRunning.IsZero() {
		pi.RunningSince = earliestRunning
	}
	return pi, nil
}

func kubectlDfMount(ctx context.Context, ns, pod, container, mount string) (used, total int64, err error) {
	inner := fmt.Sprintf("df -B1 %s 2>/dev/null | tail -1", shellQuote(mount))
	cmd := exec.CommandContext(ctx, "kubectl", "exec", "-n", ns, pod, "-c", container, "--", "sh", "-ec", inner)
	cmd.Env = controller.KubeconfigEnv()
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("df parse: %q", out)
	}
	total, err1 := strconv.ParseInt(fields[1], 10, 64)
	used, err2 := strconv.ParseInt(fields[2], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("df numbers: %q", out)
	}
	return used, total, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func listPodsForApp(ctx context.Context, ns, label string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", ns, "-l", label, "-o", "jsonpath={.items[*].metadata.name}")
	cmd.Env = controller.KubeconfigEnv()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	return strings.Fields(s), nil
}
