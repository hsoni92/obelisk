package blueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Chain is one chained-restore attempt. Phases fill it in order.
type Chain struct {
	Repository string
	Branch     string
	Choice     string
	Passphrase string
	Files      map[string][]byte
	Bundle     Bundle
	LocalID    string
	RemoteID   string
	Mismatch   bool
}

// ChainPull fetches the bundle and records a machine-id mismatch.
// A mismatch without adopt/keep stops the chain before unlock.
func (rt *Runtime) ChainPull(ctx context.Context, ch *Chain) error {
	current, err := config.Load(rt.ConfigPath)
	if err != nil {
		return err
	}
	repo := strings.TrimSpace(ch.Repository)
	if repo == "" {
		repo = current.Blueprint.GitHub.Repository
	}
	if repo == "" {
		tok, err := rt.Tokens.Load()
		if err != nil || tok.Login == "" {
			return fmt.Errorf("github login required")
		}
		repo = tok.Login + "/" + obelisk.DefaultBlueprintRepoName
	}
	branch := current.Blueprint.GitHub.Branch
	if branch == "" {
		branch = "main"
	}
	files, err := rt.Git.Fetch(ctx, repo, branch)
	if err != nil {
		return err
	}
	var manifest Manifest
	if err := json.Unmarshal(files[FileManifest], &manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	ch.Repository = repo
	ch.Branch = branch
	ch.Files = files
	ch.LocalID = strings.TrimSpace(current.Machine.ID)
	ch.RemoteID = strings.TrimSpace(manifest.MachineID)
	ch.Mismatch = ch.LocalID != "" && ch.RemoteID != "" && ch.LocalID != ch.RemoteID
	if ch.Mismatch && ch.Choice != "adopt" && ch.Choice != "keep" {
		return ErrMachineChoice
	}
	return nil
}

// ChainUnlock decrypts the pulled bundle and holds the key in memory.
func (rt *Runtime) ChainUnlock(ch *Chain) error {
	ch.Passphrase = strings.TrimSpace(ch.Passphrase)
	if ch.Passphrase == "" {
		return fmt.Errorf("passphrase is required")
	}
	var manifest Manifest
	if err := json.Unmarshal(ch.Files[FileManifest], &manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	kdf, err := KDFFromManifest(manifest)
	if err != nil {
		return err
	}
	key, err := DeriveKey(ch.Passphrase, kdf)
	if err != nil {
		return err
	}
	if manifest.KeyID != "" && KeyID(key) != manifest.KeyID {
		return ErrWrongPassphrase
	}
	bundle, err := Unpack(ch.Files, key)
	if err != nil {
		return err
	}
	ch.Bundle = bundle
	rt.Session.Set(key, kdf)
	ch.Passphrase = ""
	return nil
}

// ChainApply writes the decrypted spec and sidecars. Funnel is forced off later.
func (rt *Runtime) ChainApply(ch *Chain) error {
	applied := ch.Bundle.Spec
	MergeInstanceSecrets(&applied, ch.Bundle.Secrets)
	if ch.Choice == "keep" && ch.LocalID != "" {
		applied.Machine.ID = ch.LocalID
	} else if ch.RemoteID != "" {
		applied.Machine.ID = ch.RemoteID
	}
	key, _, kdf, ok := rt.Session.Snapshot()
	if !ok {
		return ErrLocked
	}
	applied.Blueprint.GitHub.Repository = ch.Repository
	applied.Blueprint.GitHub.Branch = ch.Branch
	applied.Blueprint.GitHub.Generation = ch.Bundle.Manifest.Generation
	applied.Blueprint.GitHub.SyncStatus = obelisk.SyncSynced
	applied.Blueprint.GitHub.LastPushError = ""
	if tok, err := rt.Tokens.Load(); err == nil && tok.Login != "" {
		applied.Blueprint.GitHub.OAuthLogin = tok.Login
	}
	storeKDF(&applied, kdf, key)
	if err := WriteSidecars(rt.ConfigPath, ch.Bundle.Secrets); err != nil {
		return err
	}
	return config.SaveQuiet(rt.ConfigPath, applied)
}

// ChainSkipData is true when the operator kept a new machine id and must not
// restore the previous bucket.
func ChainSkipData(ch *Chain) bool {
	return ch.Mismatch && ch.Choice == "keep"
}
