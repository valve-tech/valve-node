// Network-diagnostics orchestration: the stored per-target report, the
// auto-run triggers (journal error signatures via logwatch, connection
// failures via monitor snapshots), and the cooldown/singleflight gate that
// keeps a flapping node from probe-storming the target. The probes
// themselves live in ops.NetworkDiagnostics; this file only decides WHEN
// they run and remembers the last answer.
package server

import (
	"context"
	"time"

	"github.com/valve-tech/valve-node/internal/config"
	"github.com/valve-tech/valve-node/internal/logwatch"
	"github.com/valve-tech/valve-node/internal/monitor"
	"github.com/valve-tech/valve-node/internal/ops"
)

// DiagReport is one stored diagnostics run: the ladder's items in order
// (stopping at the first failure), when it ran, and what triggered it —
// "manual", "journal: <signature>", or "monitor: <condition>".
type DiagReport struct {
	At       time.Time       `json:"at"`
	Trigger  string          `json:"trigger"`
	Items    []ops.CheckItem `json:"items"`
	FailedID string          `json:"failedId,omitempty"`
}

const (
	// diagAutoCooldown rate-limits auto-triggered runs per target: a node
	// that is down emits a trigger-worthy snapshot every poll (5s) and a
	// trigger-worthy journal line potentially every restart loop — one
	// diagnosis per cooldown window is plenty, the report doesn't change.
	diagAutoCooldown = 10 * time.Minute
	// diagRunTimeout bounds an auto-run's probes (a wedged SSH target must
	// not leak run-forever goroutines).
	diagRunTimeout = 90 * time.Second
)

// diagTriggerForHit returns the auto-run trigger description for a journal
// hit, or "" when the hit shouldn't trigger diagnostics. Only error and
// critical hits trigger — warns (e.g. a transient low-peer blip) would
// fire constantly on a syncing node.
func diagTriggerForHit(h logwatch.Hit) string {
	if h.Severity != "error" && h.Severity != "critical" {
		return ""
	}
	if h.Signature != "" {
		return "journal: " + h.Signature
	}
	return "journal: " + h.Severity + " line"
}

// diagTriggerForSnapshot returns the auto-run trigger description for a
// monitor snapshot, or "" for a healthy one. An inactive service or a
// zero peer count is a failed connection in the making — exactly when the
// operator wants the ladder run for them.
func diagTriggerForSnapshot(s monitor.Snapshot) string {
	switch {
	case !s.ExecActive:
		return "monitor: execution service inactive"
	case !s.BeaconActive:
		return "monitor: beacon service inactive"
	case s.ExecPeers == 0:
		return "monitor: execution client has 0 peers"
	case s.BeaconPeers == 0:
		return "monitor: beacon client has 0 peers"
	default:
		return ""
	}
}

// tryBeginDiag reserves the target's diagnostics slot: at most one run at
// a time, and auto runs additionally rate-limited by diagAutoCooldown
// (manual runs bypass the cooldown — the operator clicked, they get a
// fresh answer). The caller MUST pair a true return with endDiag.
func (e *targetEntry) tryBeginDiag(now time.Time, auto bool) bool {
	e.diagMu.Lock()
	defer e.diagMu.Unlock()
	if e.diagBusy {
		return false
	}
	if auto && now.Sub(e.diagLast) < diagAutoCooldown {
		return false
	}
	e.diagBusy = true
	e.diagLast = now
	return true
}

// endDiag releases the diagnostics slot; a nil report (the run errored)
// keeps the previous one.
func (e *targetEntry) endDiag(r *DiagReport) {
	e.diagMu.Lock()
	defer e.diagMu.Unlock()
	e.diagBusy = false
	if r != nil {
		e.diagLatest = r
	}
}

func (e *targetEntry) latestDiag() *DiagReport {
	e.diagMu.Lock()
	defer e.diagMu.Unlock()
	return e.diagLatest
}

// runDiagnostics runs the ops ladder for t and packages the result. It
// does NOT touch the gate — callers own tryBeginDiag/endDiag.
func (s *Server) runDiagnostics(ctx context.Context, t config.Target, trigger string) (*DiagReport, error) {
	ex, err := s.getExecutor(t)
	if err != nil {
		return nil, err
	}
	var opts ops.DiagnoseOpts
	if t.Mode == "ssh" && t.SSH != nil {
		opts.SSHMode = true
		opts.SSHHost = t.SSH.Host
	}
	items, err := ops.NetworkDiagnostics(ctx, ex, *t.Wire, opts)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ops.CheckItem{}
	}
	report := &DiagReport{At: time.Now(), Trigger: trigger, Items: items}
	if n := len(items); n > 0 && items[n-1].Status == "fail" {
		report.FailedID = items[n-1].ID
	}
	return report, nil
}

// maybeAutoDiag starts a background diagnostics run for t if trigger is
// non-empty and the gate allows it. Errors are swallowed: an auto-run is
// best-effort background work with nobody to report to — the next trigger
// after the cooldown retries.
func (s *Server) maybeAutoDiag(t config.Target, trigger string) {
	if trigger == "" || t.Wire == nil {
		return
	}
	entry := s.reg.get(t.ID)
	if !entry.tryBeginDiag(time.Now(), true) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), diagRunTimeout)
		defer cancel()
		report, err := s.runDiagnostics(ctx, t, trigger)
		if err != nil {
			entry.endDiag(nil)
			return
		}
		entry.endDiag(report)
	}()
}

// watchMonitorForDiag feeds t's monitor snapshots into the auto-trigger
// until ctx (the monitor's own lifetime) ends.
func (s *Server) watchMonitorForDiag(ctx context.Context, t config.Target, mon *monitor.Monitor) {
	ch, cancel := mon.Subscribe()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-ch:
			if !ok {
				return
			}
			s.maybeAutoDiag(t, diagTriggerForSnapshot(snap))
		}
	}
}

// watchLogsForDiag feeds t's journal hits into the auto-trigger until ctx
// (the watcher's own lifetime) ends.
func (s *Server) watchLogsForDiag(ctx context.Context, t config.Target, watch *logwatch.Watcher) {
	ch, cancel := watch.Subscribe()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case hit, ok := <-ch:
			if !ok {
				return
			}
			s.maybeAutoDiag(t, diagTriggerForHit(hit))
		}
	}
}
