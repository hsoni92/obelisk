package exposure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const edgeTLSSecretName = "obelisk-edge-tls"

const edgeDomainAnnotation = "obelisk.io/edge-base-domain"

// ReconcileEdgeTLSSecret stores the wildcard edge certificate in the cluster for
// native data services (Postgres/Redis/…) to terminate TLS behind Kong passthrough.
func ReconcileEdgeTLSSecret(ctx context.Context, namespace string) error {
	if namespace == "" {
		namespace = "obelisk"
	}
	base := obelisk.PublicEdgeBaseDomain()
	if base == "" {
		return fmt.Errorf("edge base domain required")
	}
	stored, err := secretEdgeBaseDomain(ctx, namespace)
	if err != nil {
		return err
	}
	if stored == base {
		return nil
	}

	certPEM, keyPEM, err := edgeWildcardTLSCert(base)
	if err != nil {
		return fmt.Errorf("edge tls cert: %w", err)
	}
	combined := strings.TrimSpace(certPEM) + "\n" + strings.TrimSpace(keyPEM) + "\n"

	tmpCert, err := os.CreateTemp("", "obelisk-edge-*.crt")
	if err != nil {
		return err
	}
	defer os.Remove(tmpCert.Name())
	if _, err := tmpCert.WriteString(certPEM); err != nil {
		return err
	}
	if err := tmpCert.Close(); err != nil {
		return err
	}

	tmpKey, err := os.CreateTemp("", "obelisk-edge-*.key")
	if err != nil {
		return err
	}
	defer os.Remove(tmpKey.Name())
	if _, err := tmpKey.WriteString(keyPEM); err != nil {
		return err
	}
	if err := tmpKey.Close(); err != nil {
		return err
	}

	tmpPEM, err := os.CreateTemp("", "obelisk-edge-*.pem")
	if err != nil {
		return err
	}
	defer os.Remove(tmpPEM.Name())
	if _, err := tmpPEM.WriteString(combined); err != nil {
		return err
	}
	if err := tmpPEM.Close(); err != nil {
		return err
	}

	create := exec.CommandContext(ctx, "kubectl", "create", "secret", "generic", edgeTLSSecretName,
		"-n", namespace,
		"--from-file=tls.crt="+tmpCert.Name(),
		"--from-file=tls.key="+tmpKey.Name(),
		"--from-file=tls.pem="+tmpPEM.Name(),
		"--dry-run=client", "-o", "yaml")
	create.Env = kubeEnv()
	manifest, err := create.Output()
	if err != nil {
		if out, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("render edge tls secret: %w: %s", err, strings.TrimSpace(string(out.Stderr)))
		}
		return err
	}

	annotate := exec.CommandContext(ctx, "kubectl", "annotate", "--local", "-f", "-",
		edgeDomainAnnotation+"="+base, "--dry-run=client", "-o", "yaml")
	annotate.Env = kubeEnv()
	annotate.Stdin = bytes.NewReader(manifest)
	annotated, err := annotate.Output()
	if err != nil {
		if out, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("annotate edge tls secret: %w: %s", err, strings.TrimSpace(string(out.Stderr)))
		}
		return err
	}

	apply := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	apply.Env = kubeEnv()
	apply.Stdin = bytes.NewReader(annotated)
	out, err := apply.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply edge tls secret: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func secretEdgeBaseDomain(ctx context.Context, namespace string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "secret", edgeTLSSecretName,
		"-n", namespace, "-o", fmt.Sprintf("jsonpath={.metadata.annotations['%s']}", edgeDomainAnnotation))
	cmd.Env = kubeEnv()
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) == 0 {
			if exit, ok := err.(*exec.ExitError); ok {
				msg = strings.TrimSpace(string(exit.Stderr))
			}
		}
		if strings.Contains(msg, "NotFound") || strings.Contains(msg, "not found") {
			return "", nil
		}
		return "", fmt.Errorf("get edge tls secret: %w: %s", err, msg)
	}
	return string(out), nil
}