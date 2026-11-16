package restore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestChainPhaseOrder(t *testing.T) {
	var got []string
	o := New()
	st := o.Run(context.Background(), Hooks{
		Pull:           func(context.Context) error { got = append(got, PhasePull); return nil },
		Unlock:         func(context.Context) error { got = append(got, PhaseUnlock); return nil },
		Apply:          func(context.Context) error { got = append(got, PhaseApply); return nil },
		RestoreData:    func(context.Context) (int, error) { got = append(got, PhaseData); return 2, nil },
		Reconcile:      func(context.Context) error { got = append(got, PhaseReconcile); return nil },
		ForceFunnelOff: func(context.Context) error { got = append(got, PhaseFunnel); return nil },
	})
	want := []string{PhasePull, PhaseUnlock, PhaseApply, PhaseData, PhaseReconcile, PhaseFunnel}
	if len(got) != len(want) {
		t.Fatalf("phases = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("phases = %v", got)
		}
	}
	if st.Phase != PhaseComplete || !st.SettingsRestored || !st.DataRestored || st.DataObjects != 2 {
		t.Fatalf("state = %+v", st)
	}
}

func TestStartSurvivesCallerCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	o := New()
	o.Start(ctx, Hooks{
		Pull: func(ctx context.Context) error {
			return ctx.Err()
		},
	})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := o.State()
		if st.Running {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if st.Phase != PhaseComplete || st.Error != "" {
			t.Fatalf("state = %+v", st)
		}
		return
	}
	t.Fatalf("state = %+v", o.State())
}

func TestChainStopsBeforeDataOnMachineChoice(t *testing.T) {
	dataCalled := false
	o := New()
	st := o.Run(context.Background(), Hooks{
		Pull:   func(context.Context) error { return nil },
		Unlock: func(context.Context) error { return nil },
		Apply: func(context.Context) error {
			return errors.New("machine id differs from blueprint; choose adopt or keep")
		},
		RestoreData: func(context.Context) (int, error) {
			dataCalled = true
			return 1, nil
		},
		Reconcile:      func(context.Context) error { return nil },
		ForceFunnelOff: func(context.Context) error { return nil },
	})
	if dataCalled {
		t.Fatal("data restore ran after machine-id failure")
	}
	if st.Phase != PhaseFailed || st.SettingsRestored {
		t.Fatalf("state = %+v", st)
	}
}
