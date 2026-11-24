package hostidentity

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func cloudAnchor(ctx context.Context) (string, bool) {
	if anchor := strings.TrimSpace(os.Getenv("OBELISK_MACHINE_ANCHOR")); anchor != "" {
		return anchor, true
	}
	if id := metadataGET(ctx, "http://169.254.169.254/latest/meta-data/instance-id", nil); id != "" {
		return "aws:" + id, true
	}
	if id := metadataGET(ctx, "http://metadata.google.internal/computeMetadata/v1/instance/id",
		map[string]string{"Metadata-Flavor": "Google"}); id != "" {
		return "gcp:" + id, true
	}
	if id := metadataGET(ctx, "http://169.254.169.254/metadata/v1/id", nil); id != "" {
		return "do:" + id, true
	}
	if id := metadataGET(ctx, "http://169.254.169.254/metadata/v1/instance-id", nil); id != "" {
		return "openstack:" + id, true
	}
	return "", false
}

func metadataGET(ctx context.Context, url string, headers map[string]string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// DiscoverAnchor returns a stable host label for machine id derivation.
func DiscoverAnchor() (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if anchor, ok := cloudAnchor(ctx); ok {
		return anchor, true
	}
	return dmiProductUUID()
}
