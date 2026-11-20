package controller

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/exposure"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const (
	defaultWorkers        = 4
	defaultAttemptTimeout = 3 * time.Minute
)

type Scheduler struct {
	rec             Reconciler
	status          *statusStore
	workers         int
	attemptTimeout  time.Duration
	scanCh          chan struct{}
	instCh          chan string
	mu              sync.Mutex
	instLocks       map[string]*sync.Mutex
	running         map[string]bool
	reschedule      map[string]uint64
	exposureErr     string
	exposureErrTime time.Time
}

func NewScheduler(r Reconciler) *Scheduler {
	return &Scheduler{
		rec:            r,
		status:         newStatusStore(),
		workers:        defaultWorkers,
		attemptTimeout: defaultAttemptTimeout,
		scanCh:         make(chan struct{}, 1),
		instCh:         make(chan string, 64),
		instLocks:      map[string]*sync.Mutex{},
		running:        map[string]bool{},
		reschedule:     map[string]uint64{},
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.dispatch(ctx)
	for i := 0; i < s.workers; i++ {
		go s.worker(ctx)
	}
}

func (s *Scheduler) EnqueueFullScan() {
	select {
	case s.scanCh <- struct{}{}:
	default:
	}
}

func (s *Scheduler) EnqueueInstance(stateKey string) {
	select {
	case s.instCh <- stateKey:
	default:
		s.EnqueueFullScan()
	}
}

func (s *Scheduler) InstanceStatus(key string) *InstanceStatus {
	return s.status.get(key)
}

func (s *Scheduler) ExposureError() (string, time.Time) {
	return s.exposureErr, s.exposureErrTime
}

func (s *Scheduler) ObserveStates(ctx context.Context, spec obelisk.Spec) map[string]string {
	out := map[string]string{}
	for _, inst := range spec.Instances {
		key := inst.StateKey()
		if st := s.status.get(key); st != nil && st.State != "" {
			out[key] = st.State
			continue
		}
		cr := s.rec.classifyInstance(ctx, inst)
		out[key] = cr.State
	}
	return out
}

func (s *Scheduler) dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.scanCh:
			s.runScan(ctx)
		}
	}
}

func (s *Scheduler) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case key := <-s.instCh:
			s.runInstanceKeyed(ctx, key)
		}
	}
}

func (s *Scheduler) runScan(ctx context.Context) {
	spec, err := config.Load(s.rec.ConfigPath)
	if err != nil {
		slog.Warn("scheduler load config", "err", err)
		return
	}
	vpsEdge := obelisk.VPSEdge(spec)
	if !s.rec.reconcileDisabled() && vpsEdge && exposure.NeedsNativeEdgeTLS(spec) {
		if err := exposure.ReconcileEdgeTLSSecret(ctx, s.rec.Namespace); err != nil {
			slog.Warn("edge tls", "err", err)
		}
	}

	ordered := instancesReconcileOrder(spec.Instances)
	var disabled, enabled []obelisk.ServiceInstance
	for _, inst := range ordered {
		if inst.Enabled {
			enabled = append(enabled, inst)
		} else {
			disabled = append(disabled, inst)
		}
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.workers)
	run := func(inst obelisk.ServiceInstance) {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.reconcileOne(ctx, vpsEdge, spec, inst)
		}()
	}
	for _, inst := range disabled {
		run(inst)
	}
	for _, inst := range enabled {
		run(inst)
	}
	wg.Wait()

	fed, _ := FederatedTargetsForConfig(s.rec.ConfigPath, spec)
	if err := exposure.Reconcile(ctx, spec, s.rec.HelmRoot, s.rec.Namespace, fed, s.rec.ConfigPath); err != nil {
		s.exposureErr = err.Error()
		s.exposureErrTime = time.Now()
		slog.Warn("exposure reconcile", "err", err)
	} else {
		s.exposureErr = ""
	}
	if pruned := s.rec.pruneOrphanReleases(ctx, spec); len(pruned) > 0 {
		clearOrphanTracking(pruned)
		slog.Info("pruned orphan releases", "releases", pruned)
	}
}

func (s *Scheduler) runInstanceKeyed(ctx context.Context, stateKey string) {
	spec, err := config.Load(s.rec.ConfigPath)
	if err != nil {
		return
	}
	for _, inst := range spec.Instances {
		if inst.StateKey() == stateKey {
			s.reconcileOne(ctx, obelisk.VPSEdge(spec), spec, inst)
			return
		}
	}
	s.EnqueueFullScan()
}

func (s *Scheduler) instLock(key string) func() {
	s.mu.Lock()
	if s.instLocks[key] == nil {
		s.instLocks[key] = &sync.Mutex{}
	}
	m := s.instLocks[key]
	s.mu.Unlock()
	m.Lock()
	return m.Unlock
}

func (s *Scheduler) reconcileOne(parent context.Context, vpsEdge bool, spec obelisk.Spec, inst obelisk.ServiceInstance) {
	key := inst.StateKey()
	unlock := s.instLock(key)
	defer unlock()

	prev := s.status.get(key)
	if prev != nil && !prev.NextRetry.IsZero() && time.Now().Before(prev.NextRetry) {
		return
	}

	ctx, cancel := context.WithTimeout(parent, s.attemptTimeout)
	defer cancel()

	attempts := 1
	if prev != nil {
		attempts = prev.Attempts + 1
	}

	release := inst.ReleaseName()
	helmSt, _ := s.rec.helmReleaseStatus(ctx, release)
	pendingSince := time.Time{}
	if prev != nil {
		pendingSince = prev.PendingSince
	}
	if helmSt == "pending-install" && pendingSince.IsZero() {
		pendingSince = time.Now()
	}
	if helmSt != "pending-install" {
		pendingSince = time.Time{}
	}

	hash := instanceDesiredHash(vpsEdge, s.rec.edgeTLSSecretReady(ctx), inst)

	if !s.rec.reconcileDisabled() {
		if err := s.rec.safeHealHelm(ctx, release, helmSt, pendingSince); err != nil {
			slog.Warn("safe heal", "release", release, "err", err)
		}
		helmSt, _ = s.rec.helmReleaseStatus(ctx, release)
	}

	var applyErr error
	state := "starting"
	reason := ""
	detail := ""

	if s.rec.reconcileDisabled() {
		if inst.Enabled {
			state = "starting"
		} else {
			state = "stopped"
		}
	} else {
		state, reason, detail, applyErr = s.applyInstance(ctx, vpsEdge, inst, helmSt)
	}

	cr := s.rec.classifyInstance(ctx, inst)
	if applyErr != nil {
		state = "degraded"
		if strings.Contains(applyErr.Error(), "chart rendered") || strings.Contains(applyErr.Error(), "template") {
			reason = ReasonChartInvalid
		} else if ctx.Err() != nil {
			reason = ReasonApplyTimeout
		} else if helmErrorRetryable(applyErr.Error()) {
			reason = ReasonBackoffWait
		}
		detail = applyErr.Error()
	} else {
		state = cr.State
		if reason == "" {
			reason = cr.ReasonCode
		}
		if detail == "" {
			detail = cr.Detail
		}
	}

	nextRetry := time.Time{}
	if applyErr != nil && (reason == ReasonBackoffWait || reason == ReasonApplyTimeout) {
		nextRetry = time.Now().Add(backoffDuration(attempts))
	}

	st := InstanceStatus{
		State:        state,
		ReasonCode:   reason,
		Detail:       detail,
		DesiredHash:  hash,
		Attempts:     attempts,
		LastAttempt:  time.Now(),
		NextRetry:    nextRetry,
		PendingSince: pendingSince,
	}
	if state == "running" {
		st.LastSuccess = time.Now()
	}
	s.status.set(key, st)
	if applyErr != nil {
		slog.Warn("instance reconcile", "key", key, "err", applyErr)
	}
}

func (s *Scheduler) applyInstance(ctx context.Context, vpsEdge bool, inst obelisk.ServiceInstance, helmSt string) (state, reason, detail string, err error) {
	chart := fmt.Sprintf("%s/deploy/helm/services/%s", s.rec.HelmRoot, inst.Type)
	release := inst.ReleaseName()
	ns := s.rec.namespace()

	valuesPath, cleanup, err := writeInstanceValues(vpsEdge, s.rec.edgeTLSSecretReady(ctx), inst)
	if err != nil {
		return "degraded", ReasonChartInvalid, err.Error(), err
	}
	defer cleanup()

	pods := s.rec.PodsForRelease(ctx, release)
	needsApply, drift := s.rec.releaseNeedsHelmApply(ctx, helmSt, valuesPath, release, pods)

	if !inst.Enabled {
		if !needsApply && helmSt == "deployed" && pods.Total == 0 {
			return "stopped", "", "", nil
		}
		if err := s.rec.helmTemplatePreflight(ctx, chart, release, ns, valuesPath); err != nil {
			return "degraded", ReasonChartInvalid, err.Error(), err
		}
		opts := helmUpgradeOpts{waitTimeout: 2 * time.Minute}
		if err := s.rec.helmUpgradeInstallOnce(ctx, inst.Type, chart, release, ns, valuesPath, opts); err != nil {
			return "degraded", ReasonBackoffWait, err.Error(), err
		}
		cr := s.rec.classifyInstance(ctx, inst)
		return cr.State, cr.ReasonCode, cr.Detail, nil
	}

	if helmSt == "deployed" && !needsApply && pods.Ready == pods.Total && pods.Total > 0 {
		if err := s.rec.ensureInstanceDatabases(ctx, inst); err != nil {
			if execAccessDenied(err) {
				cr := s.rec.classifyInstance(ctx, inst)
				return cr.State, ReasonDBInitFailed, err.Error(), nil
			}
			return "degraded", ReasonDBInitFailed, err.Error(), err
		}
		return "running", "", "", nil
	}

	if helmSt == "deployed" && !needsApply && pods.PrimaryReason != "" {
		cr := s.rec.classifyInstance(ctx, inst)
		return cr.State, cr.ReasonCode, cr.Detail, nil
	}

	if !needsApply && helmSt == "deployed" {
		cr := s.rec.classifyInstance(ctx, inst)
		return cr.State, cr.ReasonCode, cr.Detail, nil
	}

	_ = drift
	if err := s.rec.helmTemplatePreflight(ctx, chart, release, ns, valuesPath); err != nil {
		return "degraded", ReasonChartInvalid, err.Error(), err
	}
	opts := helmUpgradeOpts{waitTimeout: 2 * time.Minute}
	if err := s.rec.helmUpgradeInstallOnce(ctx, inst.Type, chart, release, ns, valuesPath, opts); err != nil {
		return "degraded", ReasonBackoffWait, err.Error(), err
	}

	cr := s.rec.classifyInstance(ctx, inst)
	if cr.State == "running" {
		if err := s.rec.ensureInstanceDatabases(ctx, inst); err != nil {
			if execAccessDenied(err) {
				return cr.State, ReasonDBInitFailed, err.Error(), nil
			}
			return "degraded", ReasonDBInitFailed, err.Error(), err
		}
	}
	return cr.State, cr.ReasonCode, cr.Detail, nil
}