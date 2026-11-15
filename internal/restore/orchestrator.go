package restore

import (
	"context"
	"sync"
	"time"
)

const (
	PhaseIdle      = "idle"
	PhasePull      = "pull_blueprint"
	PhaseUnlock    = "unlock"
	PhaseApply     = "apply_control"
	PhaseData      = "restore_data"
	PhaseReconcile = "reconcile_services"
	PhaseFunnel    = "funnel_off"
	PhaseComplete  = "complete"
	PhaseFailed    = "failed"
)

// State is the chained restore progress shown by the admin UI.
type State struct {
	Phase            string   `json:"phase"`
	SettingsRestored bool     `json:"settings_restored"`
	DataRestored     bool     `json:"data_restored"`
	DataObjects      int      `json:"data_objects"`
	Error            string   `json:"error,omitempty"`
	StartedAt        string   `json:"started_at,omitempty"`
	FinishedAt       string   `json:"finished_at,omitempty"`
	Phases           []string `json:"phases,omitempty"`
	Running          bool     `json:"running"`
}

// Hooks are the phase implementations. A failing phase stops the chain.
type Hooks struct {
	Pull           func(ctx context.Context) error
	Unlock         func(ctx context.Context) error
	Apply          func(ctx context.Context) error
	RestoreData    func(ctx context.Context) (int, error)
	Reconcile      func(ctx context.Context) error
	ForceFunnelOff func(ctx context.Context) error
}

// Orchestrator runs pull → unlock → apply → data → reconcile → funnel off.
type Orchestrator struct {
	mu      sync.Mutex
	running bool
	state   State
}

func New() *Orchestrator {
	return &Orchestrator{state: State{Phase: PhaseIdle}}
}

func (o *Orchestrator) State() State {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state
}

// Start runs the chain in the background. A second start while running returns an error state.
func (o *Orchestrator) Start(ctx context.Context, h Hooks) State {
	o.mu.Lock()
	if o.running {
		st := o.state
		st.Error = "restore already running"
		o.mu.Unlock()
		return st
	}
	o.running = true
	o.state = State{
		Phase:     PhasePull,
		Running:   true,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	o.mu.Unlock()
	go func() {
		o.Run(ctx, h)
	}()
	return o.State()
}

// Run executes phases in order on the caller goroutine.
func (o *Orchestrator) Run(ctx context.Context, h Hooks) State {
	o.mu.Lock()
	if !o.running {
		o.running = true
		o.state = State{
			Phase:     PhasePull,
			Running:   true,
			StartedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		o.running = false
		o.state.Running = false
		if o.state.FinishedAt == "" {
			o.state.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		}
		o.mu.Unlock()
	}()

	if err := o.step(ctx, PhasePull, h.Pull); err != nil {
		return o.fail(err)
	}
	if err := o.step(ctx, PhaseUnlock, h.Unlock); err != nil {
		return o.fail(err)
	}
	if err := o.step(ctx, PhaseApply, h.Apply); err != nil {
		return o.fail(err)
	}
	o.mu.Lock()
	o.state.SettingsRestored = true
	o.mu.Unlock()

	n, err := o.data(ctx, h.RestoreData)
	if err != nil {
		return o.fail(err)
	}
	o.mu.Lock()
	o.state.DataObjects = n
	o.state.DataRestored = n > 0
	o.mu.Unlock()

	if err := o.step(ctx, PhaseReconcile, h.Reconcile); err != nil {
		return o.fail(err)
	}
	if err := o.step(ctx, PhaseFunnel, h.ForceFunnelOff); err != nil {
		return o.fail(err)
	}
	o.mu.Lock()
	o.state.Phase = PhaseComplete
	o.state.Phases = append(o.state.Phases, PhaseComplete)
	o.state.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	st := o.state
	o.mu.Unlock()
	return st
}

func (o *Orchestrator) step(ctx context.Context, phase string, fn func(context.Context) error) error {
	o.mu.Lock()
	o.state.Phase = phase
	o.state.Phases = append(o.state.Phases, phase)
	o.mu.Unlock()
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

func (o *Orchestrator) data(ctx context.Context, fn func(context.Context) (int, error)) (int, error) {
	o.mu.Lock()
	o.state.Phase = PhaseData
	o.state.Phases = append(o.state.Phases, PhaseData)
	o.mu.Unlock()
	if fn == nil {
		return 0, nil
	}
	return fn(ctx)
}

func (o *Orchestrator) fail(err error) State {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state.Phase = PhaseFailed
	o.state.Error = err.Error()
	o.state.Phases = append(o.state.Phases, PhaseFailed)
	o.state.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	return o.state
}
