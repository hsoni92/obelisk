package diagnostics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func probeBunqueue(ctx context.Context, healthURL string) ([]Check, error) {
	url := strings.TrimSpace(healthURL)
	if url == "" {
		return nil, fmt.Errorf("missing bunqueue health url")
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
		return checks, fmt.Errorf("bunqueue health returned %d", resp.StatusCode)
	}
	checks[0].Detail = "health OK"
	return checks, nil
}
