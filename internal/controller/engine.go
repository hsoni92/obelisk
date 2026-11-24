package controller

import (
	"context"
	"sync/atomic"
	"time"
)

var globalEngine atomic.Pointer[Engine]

// Engine coordinates background reconciliation via the Scheduler.
type Engine struct {
	Reconciler
	Scheduler *Scheduler
}

func NewEngine(r Reconciler) *Engine {
	return &Engine{
		Reconciler: r,
		Scheduler:  NewScheduler(r),
	}
}

func SetGlobalEngine(e *Engine) {
	globalEngine.Store(e)
}

func GlobalEngine() *Engine {
	return globalEngine.Load()
}

func (e *Engine) Start(ctx context.Context, interval time.Duration) {
	e.Scheduler.Start(ctx)
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		e.Scheduler.EnqueueFullScan()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				e.Scheduler.EnqueueFullScan()
			}
		}
	}()
}

func (e *Engine) Trigger() {
	if e != nil && e.Scheduler != nil {
		e.Scheduler.EnqueueFullScan()
	}
}

func (e *Engine) TriggerInstance(stateKey string) {
	if e != nil && e.Scheduler != nil {
		e.Scheduler.EnqueueInstance(stateKey)
	}
}
