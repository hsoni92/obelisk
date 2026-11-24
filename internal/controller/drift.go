package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

func instanceDesiredHash(vpsEdge, edgeTLSReady bool, inst obelisk.ServiceInstance) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(inst.Type))
	_, _ = h.Write([]byte(inst.ID))
	if inst.Enabled {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	c := inst.Credentials
	_, _ = h.Write([]byte(c.Username))
	_, _ = h.Write([]byte(c.Password))
	_, _ = h.Write([]byte(c.Database))
	_, _ = h.Write([]byte(c.AccessKey))
	_, _ = h.Write([]byte(c.SecretKey))
	if vpsEdge && edgeTLSReady && nativeEdgeTLSInCluster(inst.Type) {
		h.Write([]byte{1})
	}
	return h.Sum64()
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (r *Reconciler) releaseAppliedValuesHash(ctx context.Context, release string) (string, error) {
	ns := r.namespace()
	out, err := commandRunner(ctx, "helm", "get", "values", release, "-n", ns, "-o", "yaml", "--all")
	if err != nil {
		if HelmReleaseNotFound(out, err) {
			return "", nil
		}
		return "", err
	}
	return canonicalYAMLHash(out)
}

func canonicalYAMLHash(yamlBytes []byte) (string, error) {
	var doc any
	if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
		return "", err
	}
	canonical, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func valuesFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return canonicalYAMLHash(data)
}

func (r *Reconciler) releaseNeedsHelmApply(ctx context.Context, helmStatus string, valuesPath, release string, pods PodSummary) (bool, string) {
	switch helmStatus {
	case "failed", "pending-install", "pending-upgrade", "pending-rollback", "uninstalling":
		return true, "helm_status:" + helmStatus
	case "", "not found":
		return true, "release_missing"
	case "deployed":
		liveHash, err := r.releaseAppliedValuesHash(ctx, release)
		if err != nil {
			return true, "values_compare:" + err.Error()
		}
		wantHash, err := valuesFileHash(valuesPath)
		if err != nil {
			return true, "values_render:" + err.Error()
		}
		if liveHash != wantHash {
			return true, "values_drift"
		}
		if pods.Total > 0 && pods.Ready < pods.Total {
			return true, "pods_not_ready"
		}
		return false, ""
	default:
		return true, "helm_status:" + helmStatus
	}
}

func (r *Reconciler) helmTemplatePreflight(ctx context.Context, chart, release, ns, valuesFile string) error {
	args := []string{"template", release, chart, "-n", ns}
	if valuesFile != "" {
		args = append(args, "-f", valuesFile)
	}
	out, err := commandRunner(ctx, "helm", args...)
	if err != nil {
		text := strings.TrimSpace(string(out))
		return fmt.Errorf("%w: %s", err, text)
	}
	if strings.Contains(string(out), "sslnegotiation") {
		return fmt.Errorf("chart rendered invalid postgres sslnegotiation server config")
	}
	return nil
}
