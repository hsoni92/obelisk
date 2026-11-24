package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func qdrantAPIKey(inst obelisk.ServiceInstance) string {
	return strings.TrimSpace(inst.Credentials.AccessKey)
}

type qdrantCreateSnapshotResp struct {
	Result struct {
		Name string `json:"name"`
	} `json:"result"`
	Status string `json:"status"`
}

func qdrantDumpK8s(ctx context.Context, ns, pod, apiKey string) ([]byte, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("qdrant api key missing in instance credentials")
	}
	return withPodPortForward(ctx, ns, pod, 6333, func(base string, client *http.Client) ([]byte, error) {
		name, err := qdrantCreateFullSnapshot(ctx, client, base, apiKey)
		if err != nil {
			return nil, err
		}
		return qdrantDownloadSnapshot(ctx, client, base, apiKey, name)
	})
}

func qdrantCreateFullSnapshot(ctx context.Context, client *http.Client, base, apiKey string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/snapshots?wait=true", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("api-key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create snapshot: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create snapshot: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed qdrantCreateSnapshotResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse snapshot response: %w", err)
	}
	name := strings.TrimSpace(parsed.Result.Name)
	if name == "" {
		return "", fmt.Errorf("create snapshot: empty snapshot name in response")
	}
	return name, nil
}

func qdrantDownloadSnapshot(ctx context.Context, client *http.Client, base, apiKey, name string) ([]byte, error) {
	snapURL := base + "/snapshots/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, snapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download snapshot: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download snapshot: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	const maxSnap = 8 << 30 // 8Gi safety cap for single-node appliance backups
	out, err := io.ReadAll(io.LimitReader(resp.Body, maxSnap+1))
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("download snapshot: empty file")
	}
	if int64(len(out)) > maxSnap {
		return nil, fmt.Errorf("download snapshot: exceeds %d byte cap", maxSnap)
	}
	return out, nil
}

func withPodPortForward(
	ctx context.Context,
	ns, pod string,
	remotePort int,
	fn func(baseURL string, client *http.Client) ([]byte, error),
) ([]byte, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	pfCtx, pfCancel := context.WithCancel(ctx)
	defer pfCancel()

	cmd := exec.CommandContext(pfCtx, "kubectl", "port-forward", "-n", ns, "pod/"+pod,
		fmt.Sprintf("%d:%d", localPort, remotePort))
	cmd.Env = kubeconfigEnv()
	var pfErr bytes.Buffer
	cmd.Stderr = &pfErr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("kubectl port-forward: %w", err)
	}
	defer func() {
		pfCancel()
		_ = cmd.Wait()
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", localPort)
	if err := waitTCPReady(ctx, addr); err != nil {
		return nil, fmt.Errorf("%w (%s)", err, strings.TrimSpace(pfErr.String()))
	}

	base := "http://" + addr
	client := &http.Client{Timeout: 4 * time.Minute}
	return fn(base, client)
}

func waitTCPReady(ctx context.Context, addr string) error {
	var d net.Dialer
	for {
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err == nil {
			conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("port-forward %s not ready: %w", addr, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
