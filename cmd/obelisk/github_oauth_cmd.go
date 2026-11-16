package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/credentials"
)

func cmdGitHub(args []string) {
	if len(args) < 2 || args[0] != "oauth" {
		fmt.Fprintln(os.Stderr, "usage: obelisk github oauth set|configure|status")
		os.Exit(1)
	}
	switch args[1] {
	case "set":
		cmdGitHubOAuthSet(args[2:])
	case "configure", "wizard":
		cmdGitHubOAuthConfigure(args[2:])
	case "status":
		cmdGitHubOAuthStatus(args[2:])
	default:
		fmt.Fprintln(os.Stderr, "usage: obelisk github oauth set|configure|status")
		os.Exit(1)
	}
}

func cmdGitHubOAuthSet(args []string) {
	fs := flag.NewFlagSet("github oauth set", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	clientID := fs.String("client-id", strings.TrimSpace(os.Getenv("OBELISK_GITHUB_CLIENT_ID")), "GitHub OAuth client ID")
	clientSecret := fs.String("client-secret", strings.TrimSpace(os.Getenv("OBELISK_GITHUB_CLIENT_SECRET")), "GitHub OAuth client secret")
	redirectURI := fs.String("redirect-uri", strings.TrimSpace(os.Getenv("OBELISK_GITHUB_REDIRECT_URI")), "redirect URI (must match GitHub app)")
	publicURL := fs.String("public-url", strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_URL")), "admin UI base URL; used to derive redirect-uri when unset")
	ns := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "Kubernetes namespace")
	skipCluster := fs.Bool("file-only", false, "only write credentials file; do not touch the cluster")
	skipFile := fs.Bool("cluster-only", false, "only apply Kubernetes secret; do not write credentials file")
	noRestart := fs.Bool("no-restart", false, "do not restart obelisk-controller after updating the secret")
	_ = fs.Parse(args)

	if *clientID == "" || *clientSecret == "" {
		fmt.Fprintln(os.Stderr, "client-id and client-secret are required (--client-id/--client-secret or OBELISK_GITHUB_CLIENT_* env)")
		os.Exit(1)
	}

	err := applyGitHubOAuth(context.Background(), githubOAuthApplyOptions{
		ConfigPath:   *cfg,
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		RedirectURI:  *redirectURI,
		PublicURL:    *publicURL,
		Namespace:    *ns,
		SkipCluster:  *skipCluster,
		SkipFile:     *skipFile,
		NoRestart:    *noRestart,
	})
	if err != nil {
		dieApply(err)
	}
}

func cmdGitHubOAuthStatus(args []string) {
	fs := flag.NewFlagSet("github oauth status", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	_ = fs.Parse(args)

	loaded := blueprint.LoadOAuthConfig(*cfg)
	credDir := credentials.CredentialsDirForConfig(*cfg)
	file, hasFile, err := credentials.ReadGitHubOAuthFile(credDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out := map[string]any{
		"configured":      loaded.Configured(),
		"client_id":       loaded.ClientID,
		"redirect_uri":    loaded.RedirectURI,
		"credentials_dir": credDir,
		"file_present":    hasFile,
	}
	if hasFile && loaded.ClientID == "" {
		out = map[string]any{
			"configured":      file.ClientID != "",
			"client_id":       file.ClientID,
			"redirect_uri":    file.RedirectURI,
			"credentials_dir": credDir,
			"file_present":    true,
			"note":            "controller in cluster prefers Secret env over file; run `obelisk github oauth set` to apply both",
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}