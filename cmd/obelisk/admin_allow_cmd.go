package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/credentials"
)

func cmdAdmin(args []string) {
	if len(args) < 2 || args[0] != "allow" {
		fmt.Fprintln(os.Stderr, "usage: obelisk admin allow add|remove|list|init|clear")
		os.Exit(1)
	}
	switch args[1] {
	case "add":
		cmdAdminAllowAdd(args[2:])
	case "remove":
		cmdAdminAllowRemove(args[2:])
	case "list":
		cmdAdminAllowList(args[2:])
	case "init":
		cmdAdminAllowInit(args[2:])
	case "clear":
		cmdAdminAllowClear(args[2:])
	default:
		fmt.Fprintln(os.Stderr, "usage: obelisk admin allow add|remove|list|init|clear")
		os.Exit(1)
	}
}

func cmdAdminAllowAdd(args []string) {
	fs := flag.NewFlagSet("admin allow add", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	login := fs.String("login", "", "GitHub login to allow (required)")
	_ = fs.Parse(args)
	if strings.TrimSpace(*login) == "" {
		fmt.Fprintln(os.Stderr, "--login is required")
		os.Exit(1)
	}
	credDir := credentials.CredentialsDirForConfig(*cfg)
	list, err := credentials.AddAdminGitHubLogin(credDir, *login)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("allowed GitHub logins: %s\n", strings.Join(list.Logins, ", "))
}

func cmdAdminAllowRemove(args []string) {
	fs := flag.NewFlagSet("admin allow remove", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	login := fs.String("login", "", "GitHub login to remove (required)")
	_ = fs.Parse(args)
	if strings.TrimSpace(*login) == "" {
		fmt.Fprintln(os.Stderr, "--login is required")
		os.Exit(1)
	}
	credDir := credentials.CredentialsDirForConfig(*cfg)
	list, removed, err := credentials.RemoveAdminGitHubLogin(credDir, *login)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !removed {
		fmt.Fprintf(os.Stderr, "login %q was not on the allowlist\n", strings.TrimSpace(*login))
		os.Exit(1)
	}
	if len(list.Logins) == 0 {
		fmt.Println("allowlist is empty: all GitHub UI logins are denied until you add one or run `obelisk admin allow clear`")
		return
	}
	fmt.Printf("allowed GitHub logins: %s\n", strings.Join(list.Logins, ", "))
}

func cmdAdminAllowList(args []string) {
	fs := flag.NewFlagSet("admin allow list", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	asJSON := fs.Bool("json", false, "print JSON")
	_ = fs.Parse(args)

	credDir := credentials.CredentialsDirForConfig(*cfg)
	list, enforced, err := credentials.ReadAdminGitHubAllowlist(credDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *asJSON {
		out := map[string]any{
			"enforced":            enforced,
			"logins":              list.Logins,
			"credentials_dir":     credDir,
			"allowlist_file":      "admin-github-allowlist.json",
			"open_when_missing":   true,
			"empty_list_denies":   true,
			"local_bearer_bypass": true,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return
	}
	if !enforced {
		fmt.Println("allowlist: not configured (ANY GitHub user may sign in; this is an install from before the closed default)")
		fmt.Printf("credentials dir: %s\n", credDir)
		fmt.Println("hint: sudo obelisk admin allow add --login YOUR_GITHUB_USER   (this also closes sign-in to everyone else)")
		return
	}
	if len(list.Logins) == 0 {
		fmt.Println("allowlist: enforced, empty (no GitHub user can sign in)")
		fmt.Println("hint: sudo obelisk admin allow add --login YOUR_GITHUB_USER")
	} else {
		fmt.Printf("allowlist: %s\n", strings.Join(list.Logins, ", "))
	}
	fmt.Printf("credentials dir: %s\n", credDir)
}

func cmdAdminAllowClear(args []string) {
	fs := flag.NewFlagSet("admin allow clear", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	_ = fs.Parse(args)
	credDir := credentials.CredentialsDirForConfig(*cfg)
	if err := credentials.RemoveAdminGitHubAllowlist(credDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("WARNING: allowlist removed: ANY GitHub user may now sign in to this admin UI.")
	fmt.Println("         To close it again: sudo obelisk admin allow add --login YOUR_GITHUB_USER")
}

// cmdAdminAllowInit makes sure the allowlist exists, empty if new. The installer runs it on a
// fresh install so no GitHub user can sign in until the operator adds one. Existing lists
// are left alone.
func cmdAdminAllowInit(args []string) {
	fs := flag.NewFlagSet("admin allow init", flag.ExitOnError)
	cfg := fs.String("config", config.ResolvePath(), "config path")
	_ = fs.Parse(args)
	credDir := credentials.CredentialsDirForConfig(*cfg)
	created, err := credentials.InitAdminGitHubAllowlist(credDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if created {
		fmt.Println("allowlist created: empty, no GitHub user can sign in until you run: sudo obelisk admin allow add --login YOUR_GITHUB_USER")
		return
	}
	fmt.Println("allowlist already exists; left unchanged")
}
