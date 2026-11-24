package vitals

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// SamplerConfig drives the background vitals collector.
type SamplerConfig struct {
	ConfigPath   string
	HelmRoot     string
	Namespace    string
	Interval     time.Duration
	DiskInterval time.Duration
}

func (sc SamplerConfig) withDefaults() SamplerConfig {
	if sc.Interval <= 0 {
		sc.Interval = time.Minute
	}
	if sc.DiskInterval <= 0 {
		sc.DiskInterval = 5 * time.Minute
	}
	return sc
}

// RunSampler ticks until ctx is done.
func RunSampler(ctx context.Context, store *Store, sc SamplerConfig) {
	sc = sc.withDefaults()
	col := Collector{Namespace: sc.Namespace}
	rec := controller.Reconciler{ConfigPath: sc.ConfigPath, HelmRoot: sc.HelmRoot, Namespace: sc.Namespace}

	var collectMu sync.Mutex
	tick := time.NewTicker(sc.Interval)
	defer tick.Stop()

	var lastDisk time.Time

	run := func(doDisk bool) {
		if !collectMu.TryLock() {
			return
		}
		defer collectMu.Unlock()
		store.SetCollecting(true)
		defer store.SetCollecting(false)

		cfg := config.EnsureReadablePath(sc.ConfigPath)
		spec, err := config.Load(cfg)
		if err != nil {
			return
		}
		states := rec.ObserveServiceStates(ctx, spec)
		meta := BuildMeta(spec, states)
		store.SetDiskQueue(diskJobsForSpec(spec))

		in := collectInput{
			Spec:   spec,
			States: states,
			Meta:   meta,
			DoDisk: doDisk,
		}
		col.CollectMinute(ctx, store, in)
	}

	// First sample soon after start.
	go func() {
		time.Sleep(3 * time.Second)
		run(false)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			doDisk := time.Since(lastDisk) >= sc.DiskInterval
			if doDisk {
				lastDisk = time.Now()
			}
			go func(disk bool) {
				run(disk)
			}(doDisk)
		}
	}
}

// SnapshotForSpec builds the API vitals block from the store and live spec.
func SnapshotForSpec(store *Store, spec obelisk.Spec, states map[string]string) *Snapshot {
	meta := BuildMeta(spec, states)
	return store.Snapshot(meta)
}

func StartSamplerLogged(ctx context.Context, store *Store, sc SamplerConfig) {
	go func() {
		log.Printf("vitals sampler: interval %s disk %s", sc.withDefaults().Interval, sc.withDefaults().DiskInterval)
		RunSampler(ctx, store, sc)
	}()
}
