package diagnostics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func probeHTTPEdge(ctx context.Context, endpoint string, headers map[string]string) ([]Check, error) {
	checks := []Check{{Name: "parse_url", OK: true}}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		checks[0].OK = false
		checks[0].Detail = "empty endpoint"
		return checks, fmt.Errorf("empty http endpoint")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, err
	}
	host := u.Hostname()
	if host == "" {
		checks = append(checks, Check{Name: "host", OK: false, Detail: "missing hostname"})
		return checks, fmt.Errorf("http endpoint missing hostname")
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		checks = append(checks, Check{Name: "request", OK: false, Detail: err.Error()})
		return checks, err
	}
	req.Host = u.Host
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		checks = append(checks, Check{Name: "http", OK: false, Detail: err.Error()})
		return checks, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	ok := resp.StatusCode >= 200 && resp.StatusCode < 500
	checks = append(checks, Check{
		Name:   "http",
		OK:     ok,
		Detail: fmt.Sprintf("HTTP %d", resp.StatusCode),
	})
	if !ok {
		return checks, fmt.Errorf("http %d", resp.StatusCode)
	}
	return checks, nil
}

func probeRustFS(ctx context.Context, endpoint, accessKey, secretKey string) ([]Check, error) {
	checks := []Check{{Name: "parse_url", OK: true}}
	endpoint = strings.TrimSpace(endpoint)
	u, err := url.Parse(endpoint)
	if err != nil {
		checks[0].OK = false
		checks[0].Detail = err.Error()
		return checks, err
	}
	host := u.Hostname()
	if host == "" {
		return append(checks, Check{Name: "host", OK: false, Detail: "missing hostname"}),
			fmt.Errorf("s3 endpoint missing hostname")
	}
	secure := u.Scheme == "https"
	if accessKey == "" || secretKey == "" {
		return probeHTTPEdge(ctx, endpoint, nil)
	}

	cli, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		checks = append(checks, Check{Name: "s3_client", OK: false, Detail: err.Error()})
		return checks, err
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	_, err = cli.ListBuckets(ctx)
	if err != nil {
		checks = append(checks, Check{Name: "list_buckets", OK: false, Detail: err.Error()})
		return checks, err
	}
	checks = append(checks, Check{Name: "list_buckets", OK: true})
	return checks, nil
}

func probeQdrant(ctx context.Context, baseURL, apiKey string) ([]Check, error) {
	url := strings.TrimRight(baseURL, "/") + "/"
	headers := map[string]string{}
	if strings.TrimSpace(apiKey) != "" {
		headers["api-key"] = apiKey
	}
	return probeHTTPEdge(ctx, url, headers)
}

func probeSearXNG(ctx context.Context, baseURL string) ([]Check, error) {
	url := strings.TrimRight(baseURL, "/") + "/"
	return probeHTTPEdge(ctx, url, nil)
}