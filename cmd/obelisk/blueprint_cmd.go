package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/internal/blueprint"
	"github.com/hsoni92/obelisk/internal/config"
)

func cmdBlueprint(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: obelisk blueprint status|unlock|lock|push|import")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("blueprint", flag.ExitOnError)
	api := fs.String("api", envAPI(), "admin API base URL")
	cfg := fs.String("config", config.ResolvePath(), "config path")
	pass := fs.String("passphrase", os.Getenv("OBELISK_BLUEPRINT_PASSPHRASE"), "blueprint passphrase")
	force := fs.Bool("force", false, "overwrite GitHub even when remote generation is ahead")
	adopt := fs.Bool("adopt-machine-id", false, "adopt machine.id from the blueprint on import")
	keep := fs.Bool("keep-machine-id", false, "keep this host's machine.id on import")
	_ = fs.Parse(args[1:])
	switch args[0] {
	case "status":
		apiCallWith(*api, *cfg, http.MethodGet, "/api/v1/blueprint/status", nil)
	case "unlock":
		if *pass == "" {
			fmt.Fprintln(os.Stderr, "passphrase is required (--passphrase or OBELISK_BLUEPRINT_PASSPHRASE)")
			os.Exit(1)
		}
		apiCallWith(*api, *cfg, http.MethodPost, "/api/v1/blueprint/unlock", map[string]string{"passphrase": *pass})
	case "lock":
		apiCallWith(*api, *cfg, http.MethodPost, "/api/v1/blueprint/lock", map[string]string{})
	case "push":
		apiCallWith(*api, *cfg, http.MethodPost, "/api/v1/blueprint/push", map[string]bool{"force": *force})
	case "import":
		if *adopt && *keep {
			fmt.Fprintln(os.Stderr, "choose only one of --adopt-machine-id or --keep-machine-id")
			os.Exit(1)
		}
		choice := ""
		if *adopt {
			choice = "adopt"
		}
		if *keep {
			choice = "keep"
		}
		apiCallWith(*api, *cfg, http.MethodPost, "/api/v1/blueprint/import", map[string]string{
			"passphrase":        *pass,
			"machine_id_choice": choice,
		})
	default:
		fmt.Fprintln(os.Stderr, "usage: obelisk blueprint status|unlock|lock|push|import")
		os.Exit(1)
	}
}

func cmdRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	chain := fs.Bool("chain", false, "run chained blueprint + data restore")
	api := fs.String("api", envAPI(), "admin API base URL")
	cfg := fs.String("config", config.ResolvePath(), "config path")
	pass := fs.String("passphrase", os.Getenv("OBELISK_BLUEPRINT_PASSPHRASE"), "blueprint passphrase")
	adopt := fs.Bool("adopt-machine-id", false, "adopt machine.id from the blueprint")
	keep := fs.Bool("keep-machine-id", false, "keep this host's machine.id and skip data restore")
	_ = fs.Parse(args)
	if !*chain {
		fmt.Fprintln(os.Stderr, "usage: obelisk restore --chain --passphrase ... [--adopt-machine-id|--keep-machine-id]")
		os.Exit(1)
	}
	if *adopt && *keep {
		fmt.Fprintln(os.Stderr, "choose only one of --adopt-machine-id or --keep-machine-id")
		os.Exit(1)
	}
	choice := ""
	if *adopt {
		choice = "adopt"
	}
	if *keep {
		choice = "keep"
	}
	body := map[string]string{"passphrase": *pass, "machine_id_choice": choice}
	apiCallWith(*api, *cfg, "POST", "/api/v1/restore/chain", body)
}

func apiCallWith(api, cfg, method, path string, body any) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(api, "/")+path, reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok, err := os.ReadFile(blueprint.LocalTokenPath(cfg)); err == nil {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tok)))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
	if resp.StatusCode >= 300 {
		os.Exit(1)
	}
}

func envAPI() string {
	if v := os.Getenv("OBELISK_API"); v != "" {
		return v
	}
	return "http://127.0.0.1:8080"
}
