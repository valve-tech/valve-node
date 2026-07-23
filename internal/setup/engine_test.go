package setup

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node/internal/executor"
)

// TestRunAll_RunsRunThenVerifyInOrderPerStep locks in RunAll's idempotent
// check-then-act shape: for each step, Verify is called first as a
// pre-check (if it already reports success, the step is considered
// already done); if it fails, Run executes, then Verify is called again to
// confirm the post-condition. All of this happens strictly in step order.
func TestRunAll_RunsRunThenVerifyInOrderPerStep(t *testing.T) {
	var calls []string
	mkStep := func(id string) Step {
		verifyCount := 0
		return Step{
			ID: id,
			Run: func(ctx context.Context, e executor.Executor, st *State) error {
				calls = append(calls, id+":run")
				return nil
			},
			Verify: func(ctx context.Context, e executor.Executor, st *State) error {
				verifyCount++
				calls = append(calls, id+":verify")
				if verifyCount == 1 {
					return fmt.Errorf("not yet satisfied")
				}
				return nil
			},
		}
	}
	steps := []Step{mkStep("a"), mkStep("b")}
	events := make(chan Event, 100)
	st := &State{Events: events}

	if err := RunAll(context.Background(), newFakeExecutor(), steps, st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}

	want := []string{
		"a:verify", "a:run", "a:verify",
		"b:verify", "b:run", "b:verify",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

// TestRunAll_VerifyErrorStopsChain: a step whose Verify never succeeds
// (even after Run) stops RunAll with an error, and no later step executes.
func TestRunAll_VerifyErrorStopsChain(t *testing.T) {
	var ran []string
	stepA := Step{
		ID: "a",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			ran = append(ran, "a:run")
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			ran = append(ran, "a:verify")
			return errors.New("a's post-condition never holds")
		},
	}
	stepB := Step{
		ID: "b",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			ran = append(ran, "b:run")
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			ran = append(ran, "b:verify")
			return nil
		},
	}
	events := make(chan Event, 100)
	st := &State{Events: events}

	err := RunAll(context.Background(), newFakeExecutor(), []Step{stepA, stepB}, st)
	if err == nil {
		t.Fatal("RunAll: want error, got nil")
	}
	for _, c := range ran {
		if strings.HasPrefix(c, "b:") {
			t.Fatalf("step b executed after step a never verified: %v", ran)
		}
	}
}

// TestRunAll_EveryEventCarriesStepID checks both step-emitted progress
// events (sent directly from within Run, as steps.go's real steps do via
// their executor Stream callback) and the engine's own completion events.
func TestRunAll_EveryEventCarriesStepID(t *testing.T) {
	events := make(chan Event, 100)
	st := &State{Events: events}

	mkStep := func(id string) Step {
		verifyCount := 0
		return Step{
			ID: id,
			Run: func(ctx context.Context, e executor.Executor, st *State) error {
				st.Events <- Event{StepID: id, Line: "working"}
				return nil
			},
			Verify: func(ctx context.Context, e executor.Executor, st *State) error {
				verifyCount++
				if verifyCount == 1 {
					return errors.New("not yet")
				}
				return nil
			},
		}
	}
	steps := []Step{mkStep("a"), mkStep("b")}

	if err := RunAll(context.Background(), newFakeExecutor(), steps, st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	close(events)

	n := 0
	for ev := range events {
		n++
		if ev.StepID == "" {
			t.Fatalf("event missing StepID: %+v", ev)
		}
	}
	if n == 0 {
		t.Fatal("no events emitted")
	}
}

// TestRunAll_MarkerProbeSkipsRun: a step whose Verify (the marker probe,
// e.g. `test -f`) already succeeds is fully skipped — Run is never called.
func TestRunAll_MarkerProbeSkipsRun(t *testing.T) {
	runCalled := false
	step := Step{
		ID: "install-x",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			runCalled = true
			_, err := e.Run(ctx, "echo installing", nil)
			return err
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			res, err := e.Run(ctx, "test -f /usr/local/bin/x", nil)
			if err != nil {
				return err
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("marker not present")
			}
			return nil
		},
	}
	e := newFakeExecutor().script("test -f /usr/local/bin/x", executor.Result{ExitCode: 0})
	events := make(chan Event, 10)
	st := &State{Events: events}

	if err := RunAll(context.Background(), e, []Step{step}, st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if runCalled {
		t.Fatal("Run was called even though the marker probe (Verify) already succeeded")
	}
	for _, c := range e.callLog() {
		if strings.Contains(c, "echo installing") {
			t.Fatalf("install command executed despite marker probe success: %v", e.callLog())
		}
	}
}

// TestRunAll_StepWithNilVerifyAlwaysRuns documents the degenerate case: a
// step with no Verify func has no marker to pre-check or confirm against,
// so Run always executes and the step is considered done once Run returns
// without error.
func TestRunAll_StepWithNilVerifyAlwaysRuns(t *testing.T) {
	runCount := 0
	step := Step{
		ID: "no-verify",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			runCount++
			return nil
		},
	}
	events := make(chan Event, 10)
	st := &State{Events: events}

	if err := RunAll(context.Background(), newFakeExecutor(), []Step{step}, st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("runCount = %d, want 1", runCount)
	}
}
