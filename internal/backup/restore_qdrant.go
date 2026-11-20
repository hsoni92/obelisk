package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func qdrantRestoreFullSnapshotK8s(ctx context.Context, ns, pod, apiKey string, snap []byte) error {
	return withPodPortForwardAction(ctx, ns, pod, 6333, func(base string, client *http.Client) error {
		return qdrantUploadFullSnapshot(ctx, client, base, apiKey, snap)
	})
}

func withPodPortForwardAction(
	ctx context.Context,
	ns, pod string,
	remotePort int,
	fn func(baseURL string, client *http.Client) error,
) error {
	_, err := withPodPortForward(ctx, ns, pod, remotePort, func(base string, client *http.Client) ([]byte, error) {
		return nil, fn(base, client)
	})
	return err
}

func qdrantUploadFullSnapshot(ctx context.Context, client *http.Client, base, apiKey string, snap []byte) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("snapshot", "restore.snapshot")
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(snap)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/snapshots/upload?wait=true", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("api-key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload snapshot: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("upload snapshot: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Status string `json:"status"`
		Result any    `json:"result"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
		if strings.EqualFold(parsed.Status, "error") {
			return fmt.Errorf("upload snapshot: %s", strings.TrimSpace(string(raw)))
		}
	}
	return nil
}