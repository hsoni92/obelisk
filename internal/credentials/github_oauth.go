package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
)

const githubOAuthFile = "github-oauth.client.json"

// GitHubOAuth holds GitHub OAuth app settings for blueprint login.
type GitHubOAuth struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

// LoginCallbackURI appends /login/callback to a public admin UI base URL.
func LoginCallbackURI(publicBase string) string {
	return strings.TrimRight(strings.TrimSpace(publicBase), "/") + "/login/callback"
}

// WriteGitHubOAuthFile writes github-oauth.client.json with mode 0600.
func WriteGitHubOAuthFile(credentialsDir string, o GitHubOAuth) (string, error) {
	if strings.TrimSpace(o.ClientID) == "" {
		return "", fmt.Errorf("client_id is required")
	}
	if strings.TrimSpace(o.ClientSecret) == "" {
		return "", fmt.Errorf("client_secret is required")
	}
	if strings.TrimSpace(o.RedirectURI) == "" {
		return "", fmt.Errorf("redirect_uri is required")
	}
	if err := os.MkdirAll(credentialsDir, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(credentialsDir, githubOAuthFile)
	data, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// ReadGitHubOAuthFile returns OAuth settings from the credentials file if present.
func ReadGitHubOAuthFile(credentialsDir string) (GitHubOAuth, bool, error) {
	path := filepath.Join(credentialsDir, githubOAuthFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return GitHubOAuth{}, false, nil
		}
		return GitHubOAuth{}, false, err
	}
	var o GitHubOAuth
	if err := json.Unmarshal(data, &o); err != nil {
		return GitHubOAuth{}, false, err
	}
	return o, true, nil
}

// ClusterGitHubOAuthOptions configures applying the in-cluster Secret.
type ClusterGitHubOAuthOptions struct {
	Namespace     string
	RestartDeploy bool
}

// ApplyClusterGitHubOAuthSecret creates or updates Secret obelisk-github-oauth.
func ApplyClusterGitHubOAuthSecret(ctx context.Context, o GitHubOAuth, opt ClusterGitHubOAuthOptions) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found on PATH")
	}
	ns := strings.TrimSpace(opt.Namespace)
	if ns == "" {
		ns = "obelisk"
	}
	args := []string{
		"create", "secret", "generic", "obelisk-github-oauth",
		"--namespace", ns,
		"--from-literal=client_id=" + o.ClientID,
		"--from-literal=client_secret=" + o.ClientSecret,
		"--from-literal=redirect_uri=" + o.RedirectURI,
		"--dry-run=client", "-o", "yaml",
	}
	create := kubectlCmd(ctx, args...)
	apply := kubectlCmd(ctx, "apply", "-f", "-")
	pipe, err := create.StdoutPipe()
	if err != nil {
		return err
	}
	apply.Stdin = pipe
	apply.Stdout = os.Stdout
	apply.Stderr = os.Stderr
	create.Stderr = os.Stderr
	if err := create.Start(); err != nil {
		return err
	}
	if err := apply.Start(); err != nil {
		_ = create.Process.Kill()
		_ = create.Wait()
		return err
	}
	createErr := create.Wait()
	applyErr := apply.Wait()
	if createErr != nil {
		return annotateKubeconfig(fmt.Errorf("kubectl create secret: %w", createErr))
	}
	if applyErr != nil {
		return annotateKubeconfig(fmt.Errorf("kubectl apply secret: %w", applyErr))
	}
	if !opt.RestartDeploy {
		return nil
	}
	restart := kubectlCmd(ctx, "rollout", "restart",
		"deployment/obelisk-controller", "--namespace", ns)
	restart.Stdout = os.Stdout
	restart.Stderr = os.Stderr
	if err := restart.Run(); err != nil {
		return annotateKubeconfig(fmt.Errorf("kubectl rollout restart: %w", err))
	}
	return nil
}

func kubectlCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = kubeconfigEnv()
	return cmd
}

func kubeconfigEnv() []string {
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("KUBECONFIG")) != "" {
		return env
	}
	if _, err := os.Stat(k3sKubeconfigPath); err == nil {
		return append(env, "KUBECONFIG="+k3sKubeconfigPath)
	}
	return env
}

func annotateKubeconfig(err error) error {
	if err == nil || os.Geteuid() == 0 {
		return err
	}
	if !strings.Contains(err.Error(), "permission denied") {
		return err
	}
	return fmt.Errorf("%w\nre-run as root: sudo obelisk configure github", err)
}

// CredentialsDirForConfig resolves where oauth and gateway credential files should live.
func CredentialsDirForConfig(configPath string) string {
	return config.CredentialsDir(configPath)
}
