package exposure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

// ReconcileKong merges instance edge routes into the platform Kong declarative config
// and applies it when it differs from the cluster ConfigMap.
func ReconcileKong(ctx context.Context, spec obelisk.Spec, helmRoot, namespace string) error {
	if namespace == "" {
		namespace = "obelisk"
	}
	basePath := filepath.Join(helmRoot, "deploy", "helm", "platform", "files", "kong-dbless.yaml")
	base, err := os.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("read kong base %s: %w", basePath, err)
	}
	rendered, err := mergeInstanceRoutes(base, spec)
	if err != nil {
		return err
	}
	current, err := kongConfigFromCluster(ctx, namespace)
	if err != nil {
		return err
	}
	if kongYAMLSemanticallyEqual(rendered, []byte(current)) {
		return nil
	}
	if err := applyKongConfig(ctx, namespace, rendered); err != nil {
		return err
	}
	return restartKong(ctx, namespace)
}

type kongDeclarative struct {
	FormatVersion string            `yaml:"_format_version"`
	Transform     bool              `yaml:"_transform"`
	Services      []kongService     `yaml:"services"`
	Certificates  []kongCertificate `yaml:"certificates,omitempty"`
}

type kongSNIName struct {
	Name string `yaml:"name"`
}

// Kong 3.4 declarative expects snis as {name: ...} objects, not bare strings.
type kongCertificate struct {
	Cert string        `yaml:"cert"`
	Key  string        `yaml:"key"`
	SNIs []kongSNIName `yaml:"snis,omitempty"`
}

type kongService struct {
	Name     string       `yaml:"name"`
	URL      string       `yaml:"url,omitempty"`
	Host     string       `yaml:"host,omitempty"`
	Port     int          `yaml:"port,omitempty"`
	Protocol string       `yaml:"protocol,omitempty"`
	Routes   []kongRoute  `yaml:"routes"`
	Plugins  []kongPlugin `yaml:"plugins,omitempty"`
}

type kongRoute struct {
	Name         string       `yaml:"name"`
	Paths        []string     `yaml:"paths,omitempty"`
	Hosts        []string     `yaml:"hosts,omitempty"`
	Protocols    []string     `yaml:"protocols,omitempty"`
	SNIs         []string     `yaml:"snis,omitempty"`
	StripPath    *bool        `yaml:"strip_path,omitempty"`
	PreserveHost *bool        `yaml:"preserve_host,omitempty"`
	Plugins      []kongPlugin `yaml:"plugins,omitempty"`
}

type kongPlugin struct {
	Name   string         `yaml:"name"`
	Config map[string]any `yaml:"config"`
}

func kongYAMLSemanticallyEqual(a, b []byte) bool {
	if strings.TrimSpace(string(a)) == "" && strings.TrimSpace(string(b)) == "" {
		return true
	}
	var da, db kongDeclarative
	if yaml.Unmarshal(a, &da) != nil {
		return false
	}
	if yaml.Unmarshal(b, &db) != nil {
		return false
	}
	return kongDeclarativeEqual(da, db)
}

func kongDeclarativeEqual(a, b kongDeclarative) bool {
	if len(a.Services) != len(b.Services) || len(a.Certificates) != len(b.Certificates) {
		return false
	}
	for i := range a.Services {
		if !kongServiceEqual(a.Services[i], b.Services[i]) {
			return false
		}
	}
	for i := range a.Certificates {
		ca, cb := a.Certificates[i], b.Certificates[i]
		if ca.Cert != cb.Cert || ca.Key != cb.Key || len(ca.SNIs) != len(cb.SNIs) {
			return false
		}
		for j := range ca.SNIs {
			if ca.SNIs[j].Name != cb.SNIs[j].Name {
				return false
			}
		}
	}
	return true
}

func kongServiceEqual(a, b kongService) bool {
	if a.Name != b.Name || a.Protocol != b.Protocol {
		return false
	}
	if a.Protocol == "tcp" {
		if a.Host != b.Host || a.Port != b.Port {
			return false
		}
	} else if a.URL != b.URL {
		return false
	}
	if len(a.Routes) != len(b.Routes) || len(a.Plugins) != len(b.Plugins) {
		return false
	}
	for i := range a.Routes {
		ra, rb := a.Routes[i], b.Routes[i]
		if ra.Name != rb.Name || !stringSliceEqual(ra.Paths, rb.Paths) || !stringSliceEqual(ra.Hosts, rb.Hosts) {
			return false
		}
		if !stringSliceEqual(ra.Protocols, rb.Protocols) || !stringSliceEqual(ra.SNIs, rb.SNIs) {
			return false
		}
		if !boolPtrEqual(ra.StripPath, rb.StripPath) || !boolPtrEqual(ra.PreserveHost, rb.PreserveHost) {
			return false
		}
	}
	for i := range a.Plugins {
		if a.Plugins[i].Name != b.Plugins[i].Name {
			return false
		}
	}
	return true
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func mergeInstanceRoutes(baseYAML []byte, spec obelisk.Spec) ([]byte, error) {
	var doc kongDeclarative
	if err := yaml.Unmarshal(baseYAML, &doc); err != nil {
		return nil, fmt.Errorf("parse kong base: %w", err)
	}
	filtered := doc.Services[:0]
	for _, svc := range doc.Services {
		if isManagedInstanceRoute(svc.Name) {
			continue
		}
		filtered = append(filtered, svc)
	}
	for _, inst := range spec.Instances {
		if !inst.Enabled {
			continue
		}
		if !obelisk.VPSEdge(spec) {
			continue
		}
		if svc, ok := kongHTTPServiceForInstance(inst); ok {
			filtered = append(filtered, svc)
		}
		if svc, ok := kongTCPServiceForInstance(inst); ok {
			filtered = append(filtered, svc)
		}
	}
	doc.Services = filtered
	// Native protocols use TLS passthrough; databases terminate TLS themselves (see edge TLS secret).
	doc.Certificates = nil
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshal kong: %w", err)
	}
	return out, nil
}

// mergeRustFSRoutes is an alias kept for tests.
func mergeRustFSRoutes(baseYAML []byte, spec obelisk.Spec) ([]byte, error) {
	return mergeInstanceRoutes(baseYAML, spec)
}

func indentPEM(pem string) string {
	return strings.TrimSpace(pem)
}

// NeedsNativeEdgeTLS reports whether any enabled instance uses the native TLS edge port.
func NeedsNativeEdgeTLS(spec obelisk.Spec) bool {
	for _, inst := range spec.Instances {
		if !inst.Enabled {
			continue
		}
		if _, ok := obelisk.NativeEdgeTypes[inst.Type]; ok {
			return true
		}
	}
	return false
}

func isManagedInstanceRoute(name string) bool {
	return strings.HasPrefix(name, "route-") || strings.HasPrefix(name, "rustfs-s3")
}

func kongHTTPServiceForInstance(inst obelisk.ServiceInstance) (kongService, bool) {
	port, ok := obelisk.HTTPEdgeTypes[inst.Type]
	if !ok {
		return kongService{}, false
	}
	hosts := obelisk.KongEdgeHosts(inst)
	if len(hosts) == 0 {
		return kongService{}, false
	}
	name := obelisk.KongRouteServiceName(inst)
	strip := false
	preserve := true
	svc := kongService{
		Name: name,
		URL:  fmt.Sprintf("http://%s:%d", inst.ReleaseName(), port),
		Routes: []kongRoute{{
			Name:         name,
			Hosts:        hosts,
			Paths:        []string{"/"},
			StripPath:    &strip,
			PreserveHost: &preserve,
		}},
	}
	if inst.Type == "rustfs" {
		svc.Plugins = []kongPlugin{
			{
				Name: "rate-limiting",
				Config: map[string]any{
					"minute": 120,
				},
			},
			{
				Name: "request-size-limiting",
				Config: map[string]any{
					"allowed_payload_size": 50,
				},
			},
		}
	}
	return svc, true
}

func kongTCPServiceForInstance(inst obelisk.ServiceInstance) (kongService, bool) {
	port, ok := obelisk.NativeEdgeTypes[inst.Type]
	if !ok {
		return kongService{}, false
	}
	snis := obelisk.KongEdgeHosts(inst)
	if len(snis) == 0 {
		return kongService{}, false
	}
	name := obelisk.KongTCPRouteServiceName(inst)
	return kongService{
		Name:     name,
		Protocol: "tcp",
		Host:     inst.ReleaseName(),
		Port:     port,
		Routes: []kongRoute{{
			Name:      name,
			Protocols: []string{"tls_passthrough"},
			SNIs:      snis,
		}},
	}, true
}

func kongConfigFromCluster(ctx context.Context, namespace string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "configmap", "kong-dbless-config",
		"-n", namespace, "-o", "jsonpath={.data.kong\\.yaml}")
	cmd.Env = kubeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "NotFound") || strings.Contains(msg, "not found") {
			return "", nil
		}
		return "", fmt.Errorf("get kong configmap: %w: %s", err, msg)
	}
	return string(out), nil
}

func applyKongConfig(ctx context.Context, namespace string, body []byte) error {
	tmp, err := os.CreateTemp("", "obelisk-kong-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	create := exec.CommandContext(ctx, "kubectl", "create", "configmap", "kong-dbless-config",
		"-n", namespace, "--from-file=kong.yaml="+tmp.Name(),
		"--dry-run=client", "-o", "yaml")
	create.Env = kubeEnv()
	manifest, err := create.Output()
	if err != nil {
		if out, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("render kong configmap: %w: %s", err, strings.TrimSpace(string(out.Stderr)))
		}
		return err
	}

	apply := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	apply.Env = kubeEnv()
	apply.Stdin = bytes.NewReader(manifest)
	out, err := apply.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply kong configmap: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func restartKong(ctx context.Context, namespace string) error {
	ok, err := deploymentExists(ctx, namespace, "obelisk-kong")
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	cmd := exec.CommandContext(ctx, "kubectl", "rollout", "restart",
		"deployment/obelisk-kong", "-n", namespace)
	cmd.Env = kubeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart kong: %w: %s", err, strings.TrimSpace(string(out)))
	}
	wait := exec.CommandContext(ctx, "kubectl", "rollout", "status",
		"deployment/obelisk-kong", "-n", namespace, "--timeout=180s")
	wait.Env = kubeEnv()
	if wout, werr := wait.CombinedOutput(); werr != nil {
		return fmt.Errorf("kong rollout: %w: %s", werr, strings.TrimSpace(string(wout)))
	}
	return nil
}

func deploymentExists(ctx context.Context, namespace, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "deployment", name, "-n", namespace)
	cmd.Env = kubeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "NotFound") || strings.Contains(msg, "not found") {
			return false, nil
		}
		return false, fmt.Errorf("get deployment %s: %w: %s", name, err, msg)
	}
	return true, nil
}

func kubeEnv() []string {
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("KUBECONFIG")) != "" {
		return env
	}
	if strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")) != "" {
		return env
	}
	if _, err := os.Stat("/etc/rancher/k3s/k3s.yaml"); err == nil {
		return append(env, "KUBECONFIG=/etc/rancher/k3s/k3s.yaml")
	}
	return env
}