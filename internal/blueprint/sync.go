package blueprint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// errContentChanged means a save landed while a push was in flight.
var errContentChanged = errors.New("blueprint changed during push")

// Syncer debounces post-save pushes and retries while crypto is unlocked.
type Syncer struct {
	ConfigPath string
	Session    *Session
	Remote     Remote
	// Debounce coalesces a burst of saves into one automatic push.
	Debounce time.Duration
	// MinInterval is the minimum gap between automatic push attempts. It keeps
	// saves from turning into GitHub API bursts (each push makes many calls).
	// Manual pushes (PushNow) are not throttled.
	MinInterval time.Duration
	// Backoff is the first retry delay after a transient failure; it doubles
	// per failure up to MaxBackoff.
	Backoff    time.Duration
	MaxBackoff time.Duration

	wake        chan struct{}
	pushMu      sync.Mutex
	specMu      sync.Mutex
	fails       int
	lastAttempt time.Time
}

func NewSyncer(configPath string, session *Session, remote Remote) *Syncer {
	return &Syncer{
		ConfigPath:  configPath,
		Session:     session,
		Remote:      remote,
		Debounce:    5 * time.Second,
		MinInterval: 15 * time.Minute,
		Backoff:     5 * time.Minute,
		MaxBackoff:  time.Hour,
		wake:        make(chan struct{}, 1),
	}
}

// autoPushWait returns how long an automatic push must still wait so that
// attempts are at least MinInterval apart. Zero means it may run now.
func (s *Syncer) autoPushWait() time.Duration {
	if s.MinInterval <= 0 || s.lastAttempt.IsZero() {
		return 0
	}
	if since := time.Since(s.lastAttempt); since < s.MinInterval {
		return s.MinInterval - since
	}
	return 0
}

// retryDelay returns the wait before retrying a failed automatic push.
// GitHub rate limits wait the full MaxBackoff; other transient errors double
// from Backoff.
func (s *Syncer) retryDelay(err error) time.Duration {
	maxDelay := s.MaxBackoff
	if maxDelay <= 0 {
		maxDelay = time.Hour
	}
	if isGitHubRateLimit(err) {
		return maxDelay
	}
	delay := s.Backoff
	if delay <= 0 {
		delay = 5 * time.Minute
	}
	for i := 1; i < s.fails && delay < maxDelay; i++ {
		delay *= 2
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// isGitHubRateLimit reports a GitHub 429, or a 403 caused by the API rate limit.
func isGitHubRateLimit(err error) bool {
	var se *StatusError
	if !errors.As(err, &se) {
		return false
	}
	return se.Code == http.StatusTooManyRequests ||
		(se.Code == http.StatusForbidden && strings.Contains(strings.ToLower(se.Body), "rate limit"))
}

// isNonRetryablePushError reports errors that will not change by retrying the
// same content (for example a 422 non-fast-forward ref update). Automatic
// pushes stop until the next save instead of repeating the same request.
func isNonRetryablePushError(err error) bool {
	var se *StatusError
	if !errors.As(err, &se) {
		return false
	}
	if isGitHubRateLimit(err) {
		return false
	}
	switch se.Code {
	case http.StatusUnprocessableEntity, http.StatusUnauthorized, http.StatusForbidden:
		return true
	}
	return false
}

// Enqueue coalesces saves into one push after the debounce window.
func (s *Syncer) Enqueue() {
	if s == nil {
		return
	}
	if s.Session == nil || !s.Session.Unlocked() {
		s.markPaused()
		return
	}
	s.Nudge()
}

func (s *Syncer) markPaused() {
	spec, err := config.Load(s.ConfigPath)
	if err != nil || spec.Blueprint.GitHub.Repository == "" {
		return
	}
	if spec.Blueprint.GitHub.SyncStatus == obelisk.SyncPushPaused {
		return
	}
	_ = s.UpdateSpec(func(cur *obelisk.Spec) {
		if cur.Blueprint.GitHub.Repository != "" && cur.Blueprint.GitHub.SyncStatus != obelisk.SyncPushPaused {
			cur.Blueprint.GitHub.SyncStatus = obelisk.SyncPushPaused
		}
	})
}

// healStaleNeverSynced clears a bogus import error left from before the first
// successful push (generation 0, no recorded commit on disk).
func (s *Syncer) healStaleNeverSynced() {
	if s == nil {
		return
	}
	_ = s.UpdateSpec(func(spec *obelisk.Spec) {
		if !localNeverSynced(*spec) {
			return
		}
		if strings.Contains(spec.Blueprint.GitHub.LastPushError, "import required") {
			spec.Blueprint.GitHub.LastPushError = ""
		}
		if s.Session != nil && s.Session.Unlocked() {
			switch spec.Blueprint.GitHub.SyncStatus {
			case obelisk.SyncPushPaused, obelisk.SyncFailed:
				spec.Blueprint.GitHub.SyncStatus = obelisk.SyncPending
			}
		}
	})
}

// Nudge asks the worker to schedule a push.
func (s *Syncer) Nudge() {
	if s == nil || s.wake == nil {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Loop runs until ctx is cancelled.
func (s *Syncer) Loop(ctx context.Context) {
	debounce := s.Debounce
	if debounce <= 0 {
		debounce = 5 * time.Second
	}
	timer := time.NewTimer(time.Hour)
	stopTimer(timer)
	pending := false
	statTick := time.NewTicker(2 * time.Second)
	defer statTick.Stop()
	var lastMod time.Time
	if st, err := os.Stat(s.ConfigPath); err == nil {
		lastMod = st.ModTime()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-statTick.C:
			st, err := os.Stat(s.ConfigPath)
			if err != nil || !st.ModTime().After(lastMod) {
				continue
			}
			lastMod = st.ModTime()
			if s.Session != nil && s.Session.Unlocked() {
				s.Nudge()
			} else {
				s.markPaused()
			}
		case <-s.wake:
			if s.Session == nil || !s.Session.Unlocked() {
				s.markPaused()
				continue
			}
			pending = true
			stopTimer(timer)
			timer.Reset(debounce)
		case <-timer.C:
			if !pending {
				continue
			}
			if wait := s.autoPushWait(); wait > 0 {
				// Too soon after the last attempt: keep the pending save and try
				// again when the window opens. Later saves coalesce into this push.
				timer.Reset(wait)
				continue
			}
			pending = false
			s.lastAttempt = time.Now()
			err := s.PushNow(ctx)
			if st, statErr := os.Stat(s.ConfigPath); statErr == nil {
				lastMod = st.ModTime()
			}
			if errors.Is(err, errContentChanged) {
				pending = true
				timer.Reset(debounce)
				continue
			}
			if err == nil || errors.Is(err, ErrImportRequired) || errors.Is(err, ErrLocked) || errors.Is(err, ErrOnboardingRequired) || errors.Is(err, ErrEmptyRepo) {
				s.fails = 0
				continue
			}
			if isNonRetryablePushError(err) {
				// Same content, same answer: wait for the next save instead of
				// repeating a request GitHub has already refused.
				s.fails = 0
				continue
			}
			s.fails++
			pending = true
			timer.Reset(s.retryDelay(err))
		}
	}
}

// PushNow encrypts the current spec and pushes one commit.
func (s *Syncer) PushNow(ctx context.Context) error {
	return s.push(ctx, false)
}

// ForcePush encrypts the current spec and pushes one commit even when the
// remote generation is ahead of this appliance.
func (s *Syncer) ForcePush(ctx context.Context) error {
	return s.push(ctx, true)
}

func (s *Syncer) push(ctx context.Context, force bool) error {
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	if s.Session != nil && s.Session.Unlocked() {
		s.healStaleNeverSynced()
	}
	if s.Session == nil || !s.Session.Unlocked() {
		_ = s.UpdateSpec(func(spec *obelisk.Spec) {
			if spec.Blueprint.GitHub.Repository != "" {
				spec.Blueprint.GitHub.SyncStatus = obelisk.SyncPushPaused
			}
		})
		return ErrLocked
	}
	spec, err := config.Load(s.ConfigPath)
	if err != nil {
		return err
	}
	if spec.Blueprint.GitHub.Repository == "" {
		return ErrOnboardingRequired
	}
	key, _, kdf, ok := s.Session.Snapshot()
	if !ok {
		return ErrLocked
	}
	storeKDF(&spec, kdf, key)
	secrets, err := CollectSecrets(s.ConfigPath, spec)
	if err != nil {
		return err
	}
	hash, err := ContentHash(spec, secrets)
	if err != nil {
		return err
	}
	branch := obelisk.EffectiveBlueprintBranch(spec)
	gen := spec.Blueprint.GitHub.Generation + 1
	if gen < 1 {
		gen = 1
	}
	files, ferr := s.Remote.Fetch(ctx, spec.Blueprint.GitHub.Repository, branch)
	if ferr != nil && !errors.Is(ferr, ErrEmptyRepo) {
		_ = s.fail(ferr)
		return ferr
	}
	if ferr == nil {
		var remote Manifest
		if err := json.Unmarshal(files[FileManifest], &remote); err != nil {
			_ = s.fail(err)
			return err
		}
		if clobber := machineIDClobber(remote.MachineID, spec.Machine.ID, localNeverSynced(spec), force); clobber != nil {
			_ = s.fail(clobber)
			return clobber
		}
		if remote.ContentSHA256 == hash {
			if spec.Blueprint.GitHub.Generation != remote.Generation || spec.Blueprint.GitHub.SyncStatus != obelisk.SyncSynced {
				_ = s.UpdateSpec(func(cur *obelisk.Spec) {
					cur.Blueprint.GitHub.Generation = remote.Generation
					cur.Blueprint.GitHub.SyncStatus = obelisk.SyncSynced
					cur.Blueprint.GitHub.LastPushError = ""
					storeKDF(cur, kdf, key)
				})
			}
			if s.diverged(hash) {
				return errContentChanged
			}
			return nil
		}
		if !force && remote.Generation > spec.Blueprint.GitHub.Generation && !localNeverSynced(spec) {
			err := ErrImportRequired
			_ = s.fail(err)
			return err
		}
		if remote.Generation >= gen {
			gen = remote.Generation + 1
		}
	}
	bundle, err := Pack(spec, secrets, key, kdf, gen)
	if err != nil {
		_ = s.fail(err)
		return err
	}
	if _, err := Unpack(bundle.Files, key); err != nil {
		_ = s.fail(err)
		return fmt.Errorf("blueprint pack self-check: %w", err)
	}
	message := fmt.Sprintf("blueprint generation %d machine %s", gen, spec.Machine.ID)
	sha, err := s.Remote.Push(ctx, spec.Blueprint.GitHub.Repository, branch, message, bundle.Files)
	if err != nil {
		_ = s.fail(err)
		return err
	}
	verifyFiles, err := s.Remote.Fetch(ctx, spec.Blueprint.GitHub.Repository, sha)
	if err != nil {
		err = fmt.Errorf("blueprint pushed at %s but verify fetch failed: %w", sha, err)
		_ = s.fail(err)
		return err
	}
	if _, err := Unpack(verifyFiles, key); err != nil {
		err = fmt.Errorf("blueprint pushed at %s but verify decrypt failed: %w", sha, err)
		_ = s.fail(err)
		return err
	}
	if err := s.UpdateSpec(func(cur *obelisk.Spec) {
		cur.Blueprint.GitHub.Generation = gen
		cur.Blueprint.GitHub.LastCommitSHA = sha
		cur.Blueprint.GitHub.LastSyncAt = time.Now().UTC().Format(time.RFC3339)
		cur.Blueprint.GitHub.SyncStatus = obelisk.SyncSynced
		cur.Blueprint.GitHub.LastPushError = ""
		cur.Blueprint.GitHub.Branch = branch
		storeKDF(cur, kdf, key)
	}); err != nil {
		return err
	}
	if s.diverged(hash) {
		return errContentChanged
	}
	return nil
}

func (s *Syncer) diverged(pushed string) bool {
	spec, err := config.Load(s.ConfigPath)
	if err != nil {
		return false
	}
	secrets, err := CollectSecrets(s.ConfigPath, spec)
	if err != nil {
		return false
	}
	hash, err := ContentHash(spec, secrets)
	if err != nil {
		return false
	}
	return hash != pushed
}

// localNeverSynced is true when this appliance has never recorded a successful push.
func localNeverSynced(spec obelisk.Spec) bool {
	gh := spec.Blueprint.GitHub
	return gh.Generation == 0 && strings.TrimSpace(gh.LastCommitSHA) == ""
}

func machineIDClobber(remoteID, localID string, neverSynced, force bool) error {
	remoteID = strings.TrimSpace(remoteID)
	localID = strings.TrimSpace(localID)
	if remoteID == "" || localID == "" || remoteID == localID {
		return nil
	}
	if neverSynced || force {
		return fmt.Errorf("%w (remote machine %s)", ErrMachineIDClobber, remoteID)
	}
	return nil
}

func (s *Syncer) fail(err error) error {
	msg := sanitize(err.Error())
	_ = s.UpdateSpec(func(spec *obelisk.Spec) {
		if errors.Is(err, ErrImportRequired) && localNeverSynced(*spec) {
			return
		}
		spec.Blueprint.GitHub.SyncStatus = obelisk.SyncFailed
		spec.Blueprint.GitHub.LastPushError = msg
	})
	return err
}

// UpdateSpec mutates blueprint metadata without enqueueing another push.
func (s *Syncer) UpdateSpec(fn func(*obelisk.Spec)) error {
	if s == nil {
		return nil
	}
	s.specMu.Lock()
	defer s.specMu.Unlock()
	spec, err := config.Load(s.ConfigPath)
	if err != nil {
		return err
	}
	fn(&spec)
	return config.SaveQuiet(s.ConfigPath, spec)
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
