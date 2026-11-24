package diagnostics

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// probeBrowserless hits /pressure, a cheap authenticated endpoint that does not launch Chromium.
func probeBrowserless(ctx context.Context, baseURL, token string) ([]Check, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("missing browserless url")
	}
	endpoint := base + "/pressure"
	if t := strings.TrimSpace(token); t != "" {
		endpoint += "?token=" + url.QueryEscape(t)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	checks := []Check{{Name: "pressure_http", OK: true, Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}}
	if resp.StatusCode != http.StatusOK {
		checks[0].OK = false
		return checks, fmt.Errorf("browserless pressure returned %d", resp.StatusCode)
	}
	checks[0].Detail = "pressure OK"
	return checks, nil
}
