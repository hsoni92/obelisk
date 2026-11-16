package vitals

import (
	"context"
	"strings"
	"testing"
)

func TestParseTopNodeLineK3s(t *testing.T) {
	// kubectl top node on k3d/k3s includes CPU% and MEMORY% columns.
	line := "k3d-obelisk-server-0   145m         3%     1050Mi          4%"
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 5 {
		t.Fatalf("fields %v", fields)
	}
	cpu, ok1 := parseCPUQuantity(fields[1])
	mem, ok2 := parseBytesQuantity(fields[3])
	if !ok1 || !ok2 || cpu != 145 || mem != 1050*1024*1024 {
		t.Fatalf("cpu=%d ok1=%v mem=%d ok2=%v", cpu, ok1, mem, ok2)
	}
}

func TestKubectlTopNodeIntegration(t *testing.T) {
	tops, err := kubectlTopPods(context.Background(), "obelisk")
	if err != nil {
		t.Skip("cluster not available:", err)
	}
	if len(tops) == 0 {
		t.Skip("no pods in obelisk namespace")
	}
	_, err = kubectlTopNode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}