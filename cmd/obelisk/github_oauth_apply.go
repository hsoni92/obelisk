package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/credentials"
)

type githubOAuthApplyOptions struct {
	ConfigPath   string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	PublicURL    string
	Namespace    string
	SkipCluster  bool
	SkipFile     bool
	NoRestart    bool
}

func (o *githubOAuthApplyOptions) resolveRedirect() (string, error) {
	redir := strings.TrimSpace(o.RedirectURI)
	if redir != "" {
		return redir, nil
	}
	pub := strings.TrimSpace(o.PublicURL)
	if pub == "" {
		return "", fmt.Errorf("redirect-uri or public-url is required")
	}
	return credentials.LoginCallbackURI(pub), nil
}

func applyGitHubOAuth(ctx context.Context, o githubOAuthApplyOptions) error {
	if strings.TrimSpace(o.ClientID) == "" || strings.TrimSpace(o.ClientSecret) == "" {
		return fmt.Errorf("client-id and client-secret are required")
	}
	if o.SkipFile && o.SkipCluster {
		return fmt.Errorf("choose at most one of --file-only and --cluster-only")
	}
	redir, err := o.resolveRedirect()
	if err != nil {
		return err
	}

	cred := credentials.GitHubOAuth{
		ClientID:     strings.TrimSpace(o.ClientID),
		ClientSecret: strings.TrimSpace(o.ClientSecret),
		RedirectURI:  redir,
	}
	credDir := credentials.CredentialsDirForConfig(o.ConfigPath)

	if !o.SkipFile {
		path, err := credentials.WriteGitHubOAuthFile(credDir, cred)
		if err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	if !o.SkipCluster {
		err := credentials.ApplyClusterGitHubOAuthSecret(ctx, cred, credentials.ClusterGitHubOAuthOptions{
			Namespace:     o.Namespace,
			RestartDeploy: !o.NoRestart,
		})
		if err != nil {
			return err
		}
		fmt.Printf("applied Secret obelisk-github-oauth in namespace %s\n", o.Namespace)
		if !o.NoRestart {
			fmt.Println("restarted deployment/obelisk-controller")
		}
	}
	fmt.Println("GitHub OAuth app must allow browser redirect (exact match):", redir)
	fmt.Println("Operators opening the UI at another host/IP need that origin/login/callback too, or use device login (Device Flow enabled on the app).")
	return nil
}

func dieApply(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}