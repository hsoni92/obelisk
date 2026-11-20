package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func probeUnimeter(ctx context.Context, healthURL string) ([]Check, error) {
	url := strings.TrimSpace(healthURL)
	if url == "" {
		return nil, fmt.Errorf("missing unimeter health url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	checks := []Check{{Name: "health_http", OK: true, Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}}
	if resp.StatusCode != http.StatusOK {
		checks[0].OK = false
		return checks, fmt.Errorf("unimeter health returned %d", resp.StatusCode)
	}
	var body struct {
		Status string `json:"status"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if strings.EqualFold(body.Status, "ok") {
		checks[0].Detail = "status ok"
		return checks, nil
	}
	checks[0].OK = false
	return checks, fmt.Errorf("unimeter health status %q", body.Status)
}