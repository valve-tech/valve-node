// Package setup is the step engine that turns a catalog.WireConfig into a
// running, handshaking execution+beacon client pair on a target reached
// through an executor.Executor (local or SSH — the engine itself never
// knows which). It is deliberately idempotent: every Step's Verify func
// doubles as its "is this already done" marker probe, so re-running a Plan
// against a target that's partway (or fully) set up only does the work
// still outstanding.
package setup

import (
	"context"
	"fmt"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// Step is one unit of setup work.
//
// Run performs the work. Verify checks whether the step's goal state
// already holds — it is used two ways by RunAll: as a pre-check (if it
// already succeeds, Run is skipped entirely — this is the step's
// idempotence marker probe, e.g. "does the binary already exist and run?")
// and, after Run executes, as a post-check confirming Run actually reached
// the goal state. Verify may be nil for a step with no meaningful marker
// (Run always executes and success is judged solely by Run returning nil).
type Step struct {
	ID, Title string
	Run       func(ctx context.Context, e executor.Executor, st *State) error
	Verify    func(ctx context.Context, e executor.Executor, st *State) error
}

// State carries the config a Plan's steps operate against and the channel
// steps report progress on.
type State struct {
	Wire catalog.WireConfig

	// Events is the progress stream. It is send-only and provided by the
	// caller as an already-buffered channel (steps and the engine send to
	// it — sizing the buffer so that never blocks meaningfully is the
	// caller's responsibility, e.g. for driving an SSE stream a reader
	// should be draining it concurrently anyway). Sends made via emit()
	// are ctx-aware: a stalled consumer never deadlocks RunAll past the
	// caller's ctx cancellation. Neither the engine nor any step ever
	// closes Events; the caller owns its lifecycle and must close it once
	// RunAll returns, if it intends to range over it.
	Events chan<- Event
}

// Event is one progress update, e.g. a streamed command output line, or a
// step's completion / failure.
type Event struct {
	StepID string `json:"stepId"`
	Line   string `json:"line,omitempty"`
	Done   bool   `json:"done,omitempty"`
	Err    string `json:"err,omitempty"`
}

// emit sends ev on st.Events if State/Events is set. Safe to call with a
// nil State or nil Events (e.g. from tests that don't care about progress
// output). The send is ctx-aware: if ctx is canceled before a stalled
// consumer accepts the send, emit gives up and returns ctx.Err() instead of
// blocking forever.
func emit(ctx context.Context, st *State, ev Event) error {
	if st == nil || st.Events == nil {
		return nil
	}
	select {
	case st.Events <- ev:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RunAll executes steps in order. For each step: if Verify is set, it is
// called first as a pre-check; success there means the step's goal state
// already holds, so Run is skipped and the step is reported done. If
// Verify's pre-check fails and the step has no Run, that failure is
// terminal — there is nothing that could have fixed it, so Verify is not
// called again. If Verify is unset or its pre-check fails and Run is set,
// Run executes; a Run error stops the chain immediately. After a
// successful Run, Verify (if set) is called again to confirm the goal
// state was actually reached — its error also stops the chain, and later
// steps never execute. Every event emitted carries the originating step's
// ID. RunAll also stops promptly, returning ctx's error, if ctx is
// canceled while an event send is stalled on a non-draining consumer.
func RunAll(ctx context.Context, e executor.Executor, steps []Step, st *State) error {
	for _, step := range steps {
		if step.Verify != nil {
			preErr := step.Verify(ctx, e, st)
			if preErr == nil {
				if err := emit(ctx, st, Event{StepID: step.ID, Done: true, Line: "already satisfied"}); err != nil {
					return err
				}
				continue
			}
			if step.Run == nil {
				if err := emit(ctx, st, Event{StepID: step.ID, Err: preErr.Error()}); err != nil {
					return err
				}
				return fmt.Errorf("setup: step %q: %w", step.ID, preErr)
			}
		}

		if step.Run != nil {
			if err := step.Run(ctx, e, st); err != nil {
				if eerr := emit(ctx, st, Event{StepID: step.ID, Err: err.Error()}); eerr != nil {
					return eerr
				}
				return fmt.Errorf("setup: step %q: %w", step.ID, err)
			}
		}

		if step.Verify != nil {
			if err := step.Verify(ctx, e, st); err != nil {
				if eerr := emit(ctx, st, Event{StepID: step.ID, Err: err.Error()}); eerr != nil {
					return eerr
				}
				return fmt.Errorf("setup: step %q: verify failed after run: %w", step.ID, err)
			}
		}

		if err := emit(ctx, st, Event{StepID: step.ID, Done: true}); err != nil {
			return err
		}
	}
	return nil
}
