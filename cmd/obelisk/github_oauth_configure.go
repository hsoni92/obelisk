package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/cli/prompt"
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/credentials"
)

func cmdGitHubOAuthConfigure(args []string) {
	fs := flag.NewFlagSet("github oauth configure", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	ns := fs.String("namespace", envOr("OBELISK_NAMESPACE", "obelisk"), "Kubernetes namespace")
	yes := fs.Bool("yes", false, "apply without confirmation prompt")
	_ = fs.Parse(args)

	if !prompt.IsInteractive() && !*yes {
		fmt.Fprintln(os.Stderr, "non-interactive shell: use flags with `obelisk github oauth set` or pass --yes with env OBELISK_GITHUB_*")
		os.Exit(1)
	}

	fmt.Println("GitHub OAuth wizard — blueprint login")
	fmt.Println("Create an OAuth App: https://github.com/settings/developers")
	fmt.Println("  • Enable Device Flow (headless VPS)")
	fmt.Println("  • Redirect URI must match the admin UI origin exactly (see docs/runbooks/github-blueprint.md)")
	fmt.Println()

	loaded := blueprint.LoadOAuthConfig(*cfg)
	if loaded.Configured() {
		fmt.Printf("Current: client_id=%s redirect_uri=%s\n\n", loaded.ClientID, loaded.RedirectURI)
	}

	reader := bufio.NewReader(os.Stdin)
	defID := strings.TrimSpace(os.Getenv("OBELISK_GITHUB_CLIENT_ID"))
	if defID == "" && loaded.ClientID != "" {
		defID = loaded.ClientID
	}
	clientID, err := prompt.ReadLine(os.Stdout, reader, "GitHub OAuth Client ID", defID)
	if err != nil || clientID == "" {
		dieApply(fmt.Errorf("client id is required"))
	}

	fmt.Fprintln(os.Stdout, "GitHub OAuth Client Secret (input hidden on TTY):")
	secret, err := prompt.ReadSecret(os.Stdout, "")
	if err != nil {
		dieApply(err)
	}
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("OBELISK_GITHUB_CLIENT_SECRET"))
	}
	if secret == "" {
		dieApply(fmt.Errorf("client secret is required"))
	}

	defURL := credentials.DefaultAdminPublicURL(context.Background())
	if defURL == "" {
		defURL = strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_URL"))
	}
	publicURL, err := prompt.ReadLine(os.Stdout, reader, "Admin UI base URL (no trailing path)", defURL)
	if err != nil || publicURL == "" {
		dieApply(fmt.Errorf("admin UI base URL is required (e.g. http://203.0.113.10)"))
	}
	redir := credentials.LoginCallbackURI(publicURL)
	fmt.Printf("\nRegister this callback URL in GitHub:\n  %s\n\n", redir)

	if !*yes {
		ok, err := prompt.Confirm(os.Stdout, reader, "Save credentials and restart the controller?", true)
		if err != nil {
			dieApply(err)
		}
		if !ok {
			fmt.Println("aborted")
			return
		}
	}

	err = applyGitHubOAuth(context.Background(), githubOAuthApplyOptions{
		ConfigPath:   *cfg,
		ClientID:     clientID,
		ClientSecret: secret,
		PublicURL:    publicURL,
		Namespace:    *ns,
	})
	if err != nil {
		dieApply(err)
	}
}
