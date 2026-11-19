package blueprint

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Status is the admin-facing blueprint view. Tokens are omitted.
type Status struct {
	Authenticated        bool   `json:"authenticated"`
	Login                string `json:"login,omitempty"`
	ClientConfigured     bool   `json:"client_configured"`
	Repository           string `json:"repository,omitempty"`
	Branch               string `json:"branch,omitempty"`
	Generation           int    `json:"generation"`
	RemoteGeneration     int    `json:"remote_generation,omitempty"`
	LastSyncAt           string `json:"last_sync_at,omitempty"`
	LastCommitSHA        string `json:"last_commit_sha,omitempty"`
	SyncStatus           string `json:"sync_status"`
	CryptoUnlocked       bool   `json:"crypto_unlocked"`
	ImportRequired       bool   `json:"import_required"`
	LastPushError        string `json:"last_push_error,omitempty"`
	MachineID            string `json:"machine_id,omitempty"`
	RemoteMachineID      string `json:"remote_machine_id,omitempty"`
	MachineIDMismatch    bool   `json:"machine_id_mismatch"`
	BundlePresent        bool   `json:"bundle_present"`
	NeedsOnboarding      bool   `json:"needs_onboarding"`
	DiscoveredRepository string `json:"discovered_repository,omitempty"`
	KeyID                string `json:"key_id,omitempty"`
	RemoteKeyID          string `json:"remote_key_id,omitempty"`
}

// Runtime is the in-process blueprint controller: OAuth, unlock, and sync.
type Runtime struct {
	ConfigPath string
	HelmRoot   string
	Session    *Session
	Tokens     *TokenStore
	OAuth      *OAuth
	Git        *GitHub
	Sync       *Syncer
	Sessions   *AdminSessions
	LocalToken string

	refreshMu sync.Mutex
	probeMu   sync.Mutex
	probe     probeCache
}

type probeCache struct {
	at         time.Time
	repo       string
	present    bool
	generation int
	machineID  string
	keyID      string
	err        string
}

func NewRuntime(configPath, helmRoot string) (*Runtime, error) {
	session := &Session{}
	tokens := NewTokenStore(configPath)
	oauth := NewOAuth(LoadOAuthConfig(configPath))
	oauth.CredentialsDir = config.CredentialsDir(configPath)
	rt := &Runtime{
		ConfigPath: configPath,
		HelmRoot:   helmRoot,
		Session:    session,
		Tokens:     tokens,
		OAuth:      oauth,
		Sessions:   NewAdminSessions(configPath),
	}
	rt.Git = &GitHub{Token: rt.AccessToken}
	rt.Sync = NewSyncer(configPath, session, rt.Git)
	tok, err := loadOrCreateLocalToken(configPath)
	if err != nil {
		return nil, err
	}
	rt.LocalToken = tok
	return rt, nil
}

// Start runs the debounced push loop and the access-token refresher.
func (rt *Runtime) Start(ctx context.Context) {
	go rt.Sync.Loop(ctx)
	go rt.refreshLoop(ctx)
}

func (rt *Runtime) refreshLoop(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = rt.AccessToken(ctx)
		}
	}
}

// AccessToken returns a usable GitHub token, refreshing when it is near expiry.
func (rt *Runtime) AccessToken(ctx context.Context) (string, error) {
	rt.refreshMu.Lock()
	defer rt.refreshMu.Unlock()
	tok, err := rt.Tokens.Load()
	if err != nil || tok.AccessToken == "" {
		return "", fmt.Errorf("github login required")
	}
	if tok.RefreshToken != "" && !tok.Expiry.IsZero() && time.Until(tok.Expiry) < 2*time.Minute {
		next, err := rt.OAuth.Refresh(ctx, tok)
		if err != nil {
			return "", err
		}
		if next.Login == "" {
			next.Login = tok.Login
		}
		if err := rt.Tokens.Save(next); err != nil {
			return "", err
		}
		tok = next
	}
	return tok.AccessToken, nil
}

// Authenticate accepts the admin session cookie or the on-box CLI token.
func (rt *Runtime) Authenticate(r *http.Request) (string, bool) {
	if rt == nil {
		return "", false
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		got := strings.TrimPrefix(auth, "Bearer ")
		if rt.LocalToken != "" && subtle.ConstantTimeCompare([]byte(got), []byte(rt.LocalToken)) == 1 {
			if tok, err := rt.Tokens.Load(); err == nil && tok.Login != "" {
				return tok.Login, true
			}
			return "local", true
		}
	}
	c, err := r.Cookie(CookieName)
	if err != nil {
		return "", false
	}
	return rt.Sessions.Lookup(c.Value)
}

// CompleteLogin stores tokens and opens an admin session. The passphrase is not involved.
func (rt *Runtime) CompleteLogin(ctx context.Context, tok Token) (sessionID string, err error) {
	if tok.Login == "" {
		rt.mustStage(tok)
		login, err := rt.Git.Login(ctx)
		if err != nil {
			return "", err
		}
		tok.Login = login
	}
	if err := rt.Tokens.Save(tok); err != nil {
		return "", err
	}
	_ = rt.Sync.UpdateSpec(func(spec *obelisk.Spec) {
		spec.Blueprint.GitHub.OAuthLogin = tok.Login
	})
	return rt.Sessions.Create(tok.Login)
}

// mustStage saves a token that has no login yet so Git.Login can read it.
func (rt *Runtime) mustStage(tok Token) {
	_ = rt.Tokens.Save(tok)
}

// Logout drops the admin session. OAuth tokens stay so a re-login can be explicit.
func (rt *Runtime) Logout(sessionID string) {
	rt.Sessions.Delete(sessionID)
}

// Onboard creates or selects the repo, unlocks a new key, and pushes generation 1.
func (rt *Runtime) Onboard(ctx context.Context, repository, branch, passphrase string) error {
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return fmt.Errorf("passphrase is required")
	}
	full, err := rt.Git.EnsureRepo(ctx, repository)
	if err != nil {
		return err
	}
	if branch == "" {
		branch = "main"
	}
	kdf, err := DefaultKDFParams()
	if err != nil {
		return err
	}
	key, err := DeriveKey(passphrase, kdf)
	if err != nil {
		return err
	}
	rt.Session.Set(key, kdf)
	login := ""
	if tok, err := rt.Tokens.Load(); err == nil {
		login = tok.Login
	}
	if err := rt.Sync.UpdateSpec(func(spec *obelisk.Spec) {
		spec.Blueprint.GitHub.Repository = full
		spec.Blueprint.GitHub.Branch = branch
		spec.Blueprint.GitHub.OAuthLogin = login
		spec.Blueprint.GitHub.SyncStatus = obelisk.SyncPending
		spec.Blueprint.GitHub.Generation = 0
		storeKDF(spec, kdf, key)
	}); err != nil {
		return err
	}
	if err := rt.Sync.PushNow(ctx); err != nil {
		return err
	}
	rt.InvalidateProbe()
	return nil
}

// Unlock derives the session key from the passphrase and the salt stored on the spec.
func (rt *Runtime) Unlock(passphrase string) error {
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return fmt.Errorf("passphrase is required")
	}
	spec, err := config.Load(rt.ConfigPath)
	if err != nil {
		return err
	}
	kdf, err := kdfFromSpec(spec)
	if err != nil {
		return err
	}
	key, err := DeriveKey(passphrase, kdf)
	if err != nil {
		return err
	}
	if spec.Blueprint.GitHub.KeyID != "" && KeyID(key) != spec.Blueprint.GitHub.KeyID {
		return ErrWrongPassphrase
	}
	rt.Session.Set(key, kdf)
	rt.Sync.healStaleNeverSynced()
	_ = rt.Sync.UpdateSpec(func(spec *obelisk.Spec) {
		switch spec.Blueprint.GitHub.SyncStatus {
		case obelisk.SyncPushPaused, obelisk.SyncLocked, obelisk.SyncFailed:
			spec.Blueprint.GitHub.SyncStatus = obelisk.SyncPending
		}
	})
	rt.Sync.Nudge()
	return nil
}

// Lock drops the in-memory key. Local saves continue with push paused.
func (rt *Runtime) Lock() error {
	rt.Session.Clear()
	return rt.Sync.UpdateSpec(func(spec *obelisk.Spec) {
		if spec.Blueprint.GitHub.Repository != "" {
			spec.Blueprint.GitHub.SyncStatus = obelisk.SyncLocked
		}
	})
}

// Import pulls the remote bundle, decrypts it, and replaces local desired state.
// choice is "adopt" or "keep" when machine ids differ.
// When passphrase is empty, an already-unlocked session key is tried first.
func (rt *Runtime) Import(ctx context.Context, passphrase, repository, choice string) error {
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" && !rt.Session.Unlocked() {
		return fmt.Errorf("passphrase is required")
	}
	current, err := config.Load(rt.ConfigPath)
	if err != nil {
		return err
	}
	repo := strings.TrimSpace(repository)
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
	var (
		kdf  KDFParams
		key  []byte
		bundle Bundle
	)
	var importErr error
	for attempt := 0; attempt < 2; attempt++ {
		files, err := rt.Git.Fetch(ctx, repo, branch)
		if err != nil {
			return err
		}
		var manifest Manifest
		if err := json.Unmarshal(files[FileManifest], &manifest); err != nil {
			return fmt.Errorf("parse manifest: %w", err)
		}
		if passphrase != "" {
			kdf, err = KDFFromManifest(manifest)
			if err != nil {
				return err
			}
			key, err = DeriveKey(passphrase, kdf)
			if err != nil {
				return err
			}
			if manifest.KeyID != "" && KeyID(key) != manifest.KeyID {
				return fmt.Errorf("%w (derived key_id %s, manifest key_id %s)", ErrWrongPassphrase, KeyID(key), manifest.KeyID)
			}
		} else {
			sKey, sKeyID, sKDF, ok := rt.Session.Snapshot()
			if !ok {
				return ErrLocked
			}
			if manifest.KeyID != "" && sKeyID != manifest.KeyID {
				return fmt.Errorf("passphrase is required to unlock remote blueprint: %w", ErrWrongPassphrase)
			}
			key = sKey
			kdf, err = KDFFromManifest(manifest)
			if err != nil {
				kdf = sKDF
			}
		}
		bundle, importErr = Unpack(files, key)
		if importErr == nil {
			break
		}
		if !errors.Is(importErr, ErrBundleCorrupt) && !errors.Is(importErr, ErrTamper) {
			return importErr
		}
		if attempt == 0 {
			rt.InvalidateProbe()
			continue
		}
		return importErr
	}
	localID := strings.TrimSpace(current.Machine.ID)
	remoteID := strings.TrimSpace(bundle.Manifest.MachineID)
	if localID != "" && remoteID != "" && localID != remoteID {
		if choice != "adopt" && choice != "keep" {
			return ErrMachineChoice
		}
	}
	applied := bundle.Spec
	MergeInstanceSecrets(&applied, bundle.Secrets)
	if choice == "keep" && localID != "" {
		applied.Machine.ID = localID
	} else if remoteID != "" {
		applied.Machine.ID = remoteID
	}
	if applied.Deployment.Profile == "" {
		applied.Deployment.Profile = current.Deployment.Profile
	}
	applied.Blueprint.GitHub.Repository = repo
	applied.Blueprint.GitHub.Branch = branch
	applied.Blueprint.GitHub.Generation = bundle.Manifest.Generation
	applied.Blueprint.GitHub.SyncStatus = obelisk.SyncSynced
	applied.Blueprint.GitHub.LastPushError = ""
	if tok, err := rt.Tokens.Load(); err == nil && tok.Login != "" {
		applied.Blueprint.GitHub.OAuthLogin = tok.Login
	}
	storeKDF(&applied, kdf, key)
	if err := WriteSidecars(rt.ConfigPath, bundle.Secrets); err != nil {
		return err
	}
	if err := config.SaveQuiet(rt.ConfigPath, applied); err != nil {
		return err
	}
	rt.Session.Set(key, kdf)
	rt.InvalidateProbe()
	return nil
}

// Status reports login, unlock, and sync. Remote generation is cached for 30s.
func (rt *Runtime) Status(ctx context.Context, login string) Status {
	spec, err := config.Load(rt.ConfigPath)
	st := Status{
		Authenticated:    login != "",
		Login:            login,
		ClientConfigured: rt.OAuth.Config.Configured(),
		SyncStatus:       obelisk.SyncIdle,
	}
	if err != nil {
		st.LastPushError = sanitize(err.Error())
		return st
	}
	gh := spec.Blueprint.GitHub
	st.Repository = gh.Repository
	st.Branch = gh.Branch
	st.Generation = gh.Generation
	st.LastSyncAt = gh.LastSyncAt
	st.LastCommitSHA = gh.LastCommitSHA
	st.SyncStatus = gh.SyncStatus
	if st.SyncStatus == "" {
		st.SyncStatus = obelisk.SyncIdle
	}
	st.CryptoUnlocked = rt.Session.Unlocked()
	st.LastPushError = gh.LastPushError
	st.MachineID = spec.Machine.ID
	st.KeyID = gh.KeyID
	displaySpec := spec
	if st.CryptoUnlocked {
		rt.Sync.healStaleNeverSynced()
		if spec2, err2 := config.Load(rt.ConfigPath); err2 == nil {
			displaySpec = spec2
			gh = spec2.Blueprint.GitHub
			st.SyncStatus = gh.SyncStatus
			st.LastPushError = gh.LastPushError
			st.Generation = gh.Generation
		}
	}
	rt.applySyncDisplay(&st, displaySpec)
	if login != "" {
		rt.attachRemote(ctx, &st, spec)
	}
	st.NeedsOnboarding = st.Repository == "" && !st.BundlePresent
	return st
}

func (rt *Runtime) applySyncDisplay(st *Status, spec obelisk.Spec) {
	if localNeverSynced(spec) && strings.Contains(st.LastPushError, "import required") {
		st.LastPushError = ""
	}
	if !st.CryptoUnlocked && st.Repository != "" && st.SyncStatus != obelisk.SyncFailed {
		st.SyncStatus = obelisk.SyncPushPaused
		return
	}
	if !st.CryptoUnlocked {
		return
	}
	switch st.SyncStatus {
	case obelisk.SyncPushPaused, obelisk.SyncFailed:
		if st.LastPushError != "" && !localNeverSynced(spec) {
			st.SyncStatus = obelisk.SyncFailed
		} else {
			st.SyncStatus = obelisk.SyncPending
		}
	}
}

func (rt *Runtime) attachRemote(ctx context.Context, st *Status, spec obelisk.Spec) {
	repo := st.Repository
	branch := st.Branch
	if branch == "" {
		branch = "main"
	}
	if repo == "" {
		if tok, err := rt.Tokens.Load(); err == nil && tok.Login != "" {
			repo = tok.Login + "/" + obelisk.DefaultBlueprintRepoName
			st.DiscoveredRepository = repo
		} else {
			return
		}
	}
	probe, ok := rt.cachedProbe(repo)
	if !ok {
		pctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		present, manifest, err := rt.Git.LookupRepo(pctx, repo, branch)
		cancel()
		probe = probeCache{at: time.Now(), repo: repo, present: present}
		if err != nil {
			probe.err = sanitize(err.Error())
		} else if present {
			var m Manifest
			if json.Unmarshal(manifest, &m) == nil {
				probe.generation = m.Generation
				probe.machineID = m.MachineID
				probe.keyID = m.KeyID
			}
		}
		rt.probeMu.Lock()
		rt.probe = probe
		rt.probeMu.Unlock()
	}
	st.BundlePresent = probe.present
	st.RemoteGeneration = probe.generation
	st.RemoteMachineID = probe.machineID
	st.RemoteKeyID = probe.keyID
	if probe.err != "" && st.LastPushError == "" {
		st.LastPushError = probe.err
	}
	if probe.present && probe.generation > spec.Blueprint.GitHub.Generation && !localNeverSynced(spec) {
		st.ImportRequired = true
	} else if !probe.present && strings.Contains(st.LastPushError, "import required") && !localNeverSynced(spec) {
		st.ImportRequired = true
	}
	if probe.machineID != "" && spec.Machine.ID != "" && probe.machineID != spec.Machine.ID {
		st.MachineIDMismatch = true
	}
	if st.Repository == "" && probe.present {
		st.DiscoveredRepository = repo
		st.ImportRequired = true
	}
}

// InvalidateProbe clears the cached remote probe so the next Status reads fresh GitHub state.
func (rt *Runtime) InvalidateProbe() {
	if rt == nil {
		return
	}
	rt.probeMu.Lock()
	rt.probe = probeCache{}
	rt.probeMu.Unlock()
}

func (rt *Runtime) cachedProbe(repo string) (probeCache, bool) {
	rt.probeMu.Lock()
	defer rt.probeMu.Unlock()
	if rt.probe.repo == repo && time.Since(rt.probe.at) < 30*time.Second {
		return rt.probe, true
	}
	return probeCache{}, false
}

func storeKDF(spec *obelisk.Spec, kdf KDFParams, key []byte) {
	spec.Blueprint.GitHub.KDFSalt = base64.StdEncoding.EncodeToString(kdf.Salt)
	spec.Blueprint.GitHub.KDFMemory = kdf.Memory
	spec.Blueprint.GitHub.KDFIterations = kdf.Iterations
	spec.Blueprint.GitHub.KDFParallelism = kdf.Parallelism
	if len(key) > 0 {
		spec.Blueprint.GitHub.KeyID = KeyID(key)
	}
}

func kdfFromSpec(spec obelisk.Spec) (KDFParams, error) {
	if spec.Blueprint.GitHub.KDFSalt == "" {
		return KDFParams{}, ErrOnboardingRequired
	}
	salt, err := base64.StdEncoding.DecodeString(spec.Blueprint.GitHub.KDFSalt)
	if err != nil {
		return KDFParams{}, fmt.Errorf("kdf salt: %w", err)
	}
	p := KDFParams{
		Memory:      spec.Blueprint.GitHub.KDFMemory,
		Iterations:  spec.Blueprint.GitHub.KDFIterations,
		Parallelism: spec.Blueprint.GitHub.KDFParallelism,
		Salt:        salt,
	}
	if p.Memory == 0 {
		p.Memory = ArgonMemoryKiB
	}
	if p.Iterations == 0 {
		p.Iterations = ArgonIterations
	}
	if p.Parallelism == 0 {
		p.Parallelism = ArgonThreads
	}
	return p, nil
}

func loadOrCreateLocalToken(configPath string) (string, error) {
	path := filepath.Join(config.CredentialsDir(configPath), "obelisk-local-api.token")
	if b, err := os.ReadFile(path); err == nil {
		s := strings.TrimSpace(string(b))
		if s != "" {
			return s, nil
		}
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	s := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(s+"\n"), 0o600); err != nil {
		return "", err
	}
	return s, nil
}

// LocalTokenPath is where the controller writes the on-box CLI credential.
func LocalTokenPath(configPath string) string {
	return filepath.Join(config.CredentialsDir(configPath), "obelisk-local-api.token")
}

// SessionCookie reads the raw session id from the request, if any.
func SessionCookie(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// ErrIs reports blueprint sentinel errors for HTTP mapping.
func ErrIs(err, target error) bool {
	return errors.Is(err, target)
}
