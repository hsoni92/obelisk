package backup

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
	"github.com/robfig/cron/v3"
)

var globalScheduler atomic.Pointer[ScheduleRunner]

// ScheduleRunner runs backup.policy.schedule and per-instance overrides via cron.
type ScheduleRunner struct {
	ConfigPath string
	HelmRoot   string
	Namespace  string

	mu   sync.Mutex
	cron *cron.Cron

	schedule string
	status   string
	lastErr  string

	inRun bool
}

// SetGlobalScheduleRunner registers the process-wide backup scheduler for status APIs.
func SetGlobalScheduleRunner(s *ScheduleRunner) {
	globalScheduler.Store(s)
}

// ReloadGlobalSchedule re-evaluates readiness and cron after credentials or DR tests change.
func ReloadGlobalSchedule() {
	if s := globalScheduler.Load(); s != nil {
		s.Reload()
	}
}

// ImplementationStatus reports scheduled-backup worker readiness for the admin API.
func ImplementationStatus() string {
	s := globalScheduler.Load()
	if s == nil {
		return "stub"
	}
	return s.statusString()
}

// ValidateCronSchedule checks a standard 5-field cron expression (UTC).
func ValidateCronSchedule(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(strings.TrimSpace(expr))
	return err
}

// NewScheduleRunner configures a cron-driven backup worker (not started until Start).
func NewScheduleRunner(configPath, helmRoot, namespace string) *ScheduleRunner {
	return &ScheduleRunner{
		ConfigPath: configPath,
		HelmRoot:   helmRoot,
		Namespace:  namespace,
		status:     "starting",
	}
}

// Start loads policy and begins evaluating the schedule until ctx is cancelled.
func (s *ScheduleRunner) Start(ctx context.Context) {
	SetGlobalScheduleRunner(s)
	s.Reload()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		if s.cron != nil {
			stopCtx := s.cron.Stop()
			s.mu.Unlock()
			<-stopCtx.Done()
			return
		}
		s.mu.Unlock()
	}()
}

// Reload re-reads obelisk.yaml and re-registers cron jobs (call after policy saves).
func (s *ScheduleRunner) Reload() {
	s.mu.Lock()
	defer s.mu.Unlock()

	ready, readyMsg, buckets, globalSched, err := s.plan()
	s.schedule = globalSched

	if err != nil {
		s.stopCronLocked()
		s.status = "invalid_schedule"
		s.lastErr = err.Error()
		log.Printf("backup schedule: %v", err)
		return
	}
	if !ready {
		s.stopCronLocked()
		s.status = "unconfigured"
		s.lastErr = readyMsg
		return
	}

	c := cron.New(cron.WithLocation(time.UTC))
	for expr, instances := range buckets {
		expr, instances := expr, instances
		if _, err := c.AddFunc(expr, func() { s.tick(instances, expr) }); err != nil {
			s.stopCronLocked()
			s.status = "invalid_schedule"
			s.lastErr = err.Error()
			log.Printf("backup schedule: cron add %q: %v", expr, err)
			return
		}
	}
	s.stopCronLocked()
	s.cron = c
	c.Start()
	s.status = "scheduled"
	s.lastErr = ""
	log.Printf("backup schedule: %d job(s) active UTC (%s global default)", len(buckets), globalSched)
}

func (s *ScheduleRunner) plan() (ready bool, readyMsg string, buckets map[string]InstanceSet, globalSched string, err error) {
	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, loadErr := config.Load(cfg)
	if loadErr != nil {
		return false, loadErr.Error(), nil, "", nil
	}
	buckets, err = scheduleBuckets(spec)
	if err != nil {
		return false, "", nil, "", err
	}
	globalSched = strings.TrimSpace(obelisk.ResolvedBackupPolicy(spec).Schedule)

	runner := Runner{ConfigPath: s.ConfigPath}
	ok, msg, rerr := runner.Ready()
	if rerr != nil {
		return false, rerr.Error(), nil, globalSched, nil
	}
	if !ok {
		return false, msg, buckets, globalSched, nil
	}
	return true, "", buckets, globalSched, nil
}

func (s *ScheduleRunner) stopCronLocked() {
	if s.cron == nil {
		return
	}
	stopCtx := s.cron.Stop()
	s.cron = nil
	go func() { <-stopCtx.Done() }()
}

func (s *ScheduleRunner) statusString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != "" {
		return s.status
	}
	return "stub"
}

func (s *ScheduleRunner) tick(only InstanceSet, cronExpr string) {
	s.mu.Lock()
	if s.inRun {
		s.mu.Unlock()
		log.Printf("backup schedule: skipping %q tick (run in progress)", cronExpr)
		return
	}
	s.inRun = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inRun = false
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	cfg := config.EnsureReadablePath(s.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		log.Printf("backup schedule: load config: %v", err)
		return
	}
	rec := controller.Reconciler{
		ConfigPath: cfg,
		HelmRoot:   s.HelmRoot,
		Namespace:  s.Namespace,
	}
	states := rec.ObserveServiceStates(ctx, spec)
	runner := Runner{ConfigPath: s.ConfigPath, InstanceStates: states}
	ok, msg, err := runner.Ready()
	if err != nil {
		log.Printf("backup schedule: ready check: %v", err)
		return
	}
	if !ok {
		log.Printf("backup schedule: not ready: %s", msg)
		return
	}

	out, err := runner.RunScheduledFor(ctx, only)
	if err != nil {
		log.Printf("backup schedule: %q run failed: %v", cronExpr, err)
		return
	}
	log.Printf(
		"backup schedule: %q completed (%d object(s), %d destination(s), %d skipped)",
		cronExpr,
		len(out.DataObjects),
		out.UploadedDestinations,
		len(out.DataSkipped),
	)
}