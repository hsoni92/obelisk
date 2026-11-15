package blueprint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Debounce   time.Duration
	Backoff    time.Duration

	wake   chan struct{}
	pushMu sync.Mutex
	specMu sync.Mutex
	fails  int
}

func NewSyncer(configPath string, session *Session, remote Remote) *Syncer {
	return &Syncer{
		ConfigPath: configPath,
		Session:    session,
		Remote:     remote,
		Debounce:   5 * time.Second,
		Backoff:    2 * time.Second,
		wake:       make(chan struct{}, 1),
	}
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
			pending = false
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
			s.fails++
			pending = true
			delay := s.Backoff
			if delay <= 0 {
				delay = 2 * time.Second
			}
			for i := 1; i < s.fails && i < 6; i++ {
				delay *= 2
			}
			timer.Reset(delay)
		}
	}
}

// PushNow encrypts the current spec and pushes one commit.
func (s *Syncer) PushNow(ctx context.Context) error {
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
	branch := spec.Blueprint.GitHub.Branch
	if branch == "" {
		branch = "main"
	}
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
		if remote.Generation > spec.Blueprint.GitHub.Generation && !localNeverSynced(spec) {
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
	message := fmt.Sprintf("blueprint generation %d machine %s", gen, spec.Machine.ID)
	sha, err := s.Remote.Push(ctx, spec.Blueprint.GitHub.Repository, branch, message, bundle.Files)
	if err != nil {
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
