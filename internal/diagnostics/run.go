package diagnostics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const probeTimeout = 35 * time.Second

// Run probes selected instances using the same connection URLs as the admin UI.
func Run(
	ctx context.Context,
	spec obelisk.Spec,
	conns obelisk.Connections,
	states map[string]string,
	refs []InstanceRef,
) RunResponse {
	started := time.Now().UTC().Format(time.RFC3339)
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(len(refs)+1)*probeTimeout)
	defer cancel()

	connByKey := map[string]obelisk.InstanceConnection{}
	for _, ic := range conns.Instances {
		connByKey[ic.Type+"/"+ic.ID] = ic
	}

	out := make([]InstanceResult, len(refs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)

	for i, ref := range refs {
		wg.Add(1)
		go func(i int, ref InstanceRef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = runOne(ctx, spec, connByKey, states, ref)
		}(i, ref)
	}
	wg.Wait()

	return RunResponse{StartedAt: started, Results: out}
}

func runOne(
	ctx context.Context,
	spec obelisk.Spec,
	connByKey map[string]obelisk.InstanceConnection,
	states map[string]string,
	ref InstanceRef,
) InstanceResult {
	start := time.Now()
	typ := strings.TrimSpace(ref.Type)
	id := strings.TrimSpace(ref.ID)
	res := InstanceResult{Type: typ, ID: id}

	inst, _, ok := obelisk.FindInstanceByTypeAndID(&spec, typ, id)
	if !ok {
		res.Skipped = true
		res.SkippedReason = "instance not found"
		res.Summary = res.SkippedReason
		res.DurationMS = elapsedMS(start)
		return res
	}
	if inst.DisplayName != "" {
		res.DisplayName = inst.DisplayName
	}

	stateKey := inst.StateKey()
	res.ClusterState = states[stateKey]
	if res.ClusterState == "" {
		if inst.Enabled {
			res.ClusterState = "unknown"
		} else {
			res.ClusterState = "stopped"
		}
	}

	if !inst.Enabled {
		res.Skipped = true
		res.SkippedReason = "instance disabled"
		res.Summary = res.SkippedReason
		res.DurationMS = elapsedMS(start)
		return res
	}

	ic, ok := connByKey[typ+"/"+id]
	if !ok {
		res.Skipped = true
		res.SkippedReason = "no connection URL published"
		res.Summary = res.SkippedReason
		res.DurationMS = elapsedMS(start)
		return res
	}
	if strings.TrimSpace(ic.PublicPath) != "" {
		res.PublicHost = ic.PublicPath
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	checks, err := probeInstance(probeCtx, inst, ic)
	res.Checks = checks
	res.DurationMS = elapsedMS(start)

	if err != nil {
		res.OK = false
		res.Summary = summarizeFailure(err, checks)
		return res
	}
	res.OK = true
	if res.ClusterState != "running" {
		res.Summary = "endpoint OK (cluster state: " + res.ClusterState + ")"
	} else {
		res.Summary = "endpoint OK"
	}
	return res
}

func probeInstance(ctx context.Context, inst obelisk.ServiceInstance, ic obelisk.InstanceConnection) ([]Check, error) {
	switch inst.Type {
	case "redis":
		if ic.Redis == "" {
			return nil, fmt.Errorf("missing redis url")
		}
		return probeRedis(ctx, ic.Redis)
	case "postgres":
		if ic.Postgres == "" {
			return nil, fmt.Errorf("missing postgres url")
		}
		return probePostgres(ctx, ic.Postgres)
	case "mysql":
		if ic.MySQL == "" {
			return nil, fmt.Errorf("missing mysql url")
		}
		return probeMySQL(ctx, ic.MySQL)
	case "mongodb":
		if ic.MongoDB == "" {
			return nil, fmt.Errorf("missing mongodb url")
		}
		return probeMongoDB(ctx, ic.MongoDB)
	case "rustfs":
		if ic.S3HTTP == "" {
			return nil, fmt.Errorf("missing s3 http url")
		}
		return probeRustFS(ctx, ic.S3HTTP, inst.Credentials.AccessKey, inst.Credentials.SecretKey)
	case "qdrant":
		if ic.QdrantHTTP == "" {
			return nil, fmt.Errorf("missing qdrant url")
		}
		return probeQdrant(ctx, ic.QdrantHTTP, inst.Credentials.AccessKey)
	case "searxng":
		if ic.SearXNGHTTP == "" {
			return nil, fmt.Errorf("missing searxng url")
		}
		return probeSearXNG(ctx, ic.SearXNGHTTP)
	case "redpanda":
		if ic.KafkaBootstrap == "" {
			return nil, fmt.Errorf("missing kafka bootstrap")
		}
		return probeRedpanda(ctx, ic.KafkaBootstrap)
	case "unimeter":
		if ic.UnimeterHealthHTTP == "" {
			return nil, fmt.Errorf("missing unimeter health url")
		}
		return probeUnimeter(ctx, ic.UnimeterHealthHTTP)
	default:
		return nil, fmt.Errorf("unsupported type %q", inst.Type)
	}
}

func summarizeFailure(err error, checks []Check) string {
	if err == nil {
		return "failed"
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "wrongpass"), strings.Contains(low, "authentication"), strings.Contains(low, "auth"):
		return "authentication failed"
	case strings.Contains(low, "connection refused"), strings.Contains(low, "connection reset"), strings.Contains(low, "i/o timeout"), strings.Contains(low, "no route"):
		return "network or edge unreachable (check hostname, port 9443/9080, firewall, SNI)"
	case strings.Contains(low, "tls"), strings.Contains(low, "ssl"), strings.Contains(low, "certificate"):
		return "TLS handshake failed"
	case strings.Contains(low, "parse"):
		return "invalid connection URL"
	default:
		for i := len(checks) - 1; i >= 0; i-- {
			if !checks[i].OK && checks[i].Detail != "" {
				return checks[i].Detail
			}
		}
		return msg
	}
}