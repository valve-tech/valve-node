// Package logwatch tails journald units live via executor.Executor,
// classifies each line against a signature table of known failure modes
// (falling back to a raw level-word severity for unrecognized error-ish
// lines), and keeps a capped ring buffer of the results — fanned out to
// subscribers (the SSE stream) the same way internal/monitor fans out
// Snapshots.
package logwatch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node/internal/executor"
)

// Hit is one classified log line.
type Hit struct {
	Unit      string    `json:"unit"`
	Line      string    `json:"line"`
	At        time.Time `json:"at"`
	Signature string    `json:"signature"` // "" = unclassified error-ish line
	Severity  string    `json:"severity"`  // info|warn|error|critical
	Explain   string    `json:"explain"`   // canned explanation, may be ""
	LearnURL  string    `json:"learnUrl,omitempty"`
}

// ringSize caps how many Hits Watcher retains. journald units tail forever,
// so an uncapped buffer would grow without bound over a long-running
// process; 1000 is generous for "recent activity" (an operator debugging a
// live incident) without the ring becoming a meaningful memory or Recent()
// scan cost.
const ringSize = 1000

// Watcher tails a fixed set of journald units for the lifetime of a
// context, classifying every line and keeping the most recent ringSize
// Hits.
type Watcher struct {
	exec  executor.Executor
	units []string

	mu   sync.Mutex
	ring []Hit // newest last; capped at ringSize

	subsMu sync.Mutex
	subs   map[chan Hit]struct{}
}

// New constructs a Watcher over units. It does not start tailing — call
// Start.
func New(e executor.Executor, units []string) *Watcher {
	return &Watcher{
		exec:  e,
		units: units,
		subs:  map[chan Hit]struct{}{},
	}
}

// Start begins tailing every configured unit in its own goroutine, each via
// `journalctl -u <unit> -f`, until ctx is canceled. Executor.Run is
// long-running for a follow; when ctx is canceled the underlying command
// exits and Run returns — a transport error at that point is expected and
// ignored, not treated as fatal.
func (w *Watcher) Start(ctx context.Context) {
	for _, u := range w.units {
		unit := u
		go w.tail(ctx, unit)
	}
}

// tailBackoffMin/Max bound the delay between re-invoking journalctl after
// Run returns for any reason other than ctx cancellation (SSH transport
// drop, journald restart) — doubling from tailBackoffMin up to
// tailBackoffMax. tailBackoffResetAfter: a Run that stayed up at least that
// long is treated as having recovered, so the next retry starts back at
// tailBackoffMin rather than continuing to climb toward the cap.
const (
	tailBackoffMin        = 1 * time.Second
	tailBackoffMax        = 10 * time.Second
	tailBackoffResetAfter = 30 * time.Second
)

func (w *Watcher) tail(ctx context.Context, unit string) {
	// -n 0: don't replay backlog, only new lines from "now". -o cat: raw
	// message text only, no journald metadata prefix — classification
	// works directly on the client's own log line.
	cmd := fmt.Sprintf("journalctl -u %s -f -n 0 --no-pager -o cat", shQuote(unit))

	backoff := tailBackoffMin
	for ctx.Err() == nil {
		start := time.Now()
		_, _ = w.exec.Run(ctx, cmd, &executor.RunOpts{
			Stream: func(line string) { w.handleLine(unit, line) },
		})
		if ctx.Err() != nil {
			// Canceled — Run returning is expected, not a failure to retry.
			return
		}

		// A Run that survived a while before dropping is treated as a
		// transient blip on an otherwise-healthy tail, not a persistent
		// failure — don't keep climbing the backoff for it.
		if time.Since(start) > tailBackoffResetAfter {
			backoff = tailBackoffMin
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > tailBackoffMax {
			backoff = tailBackoffMax
		}
	}
}

func (w *Watcher) handleLine(unit, line string) {
	hit, ok := classify(unit, line, time.Now())
	if !ok {
		return
	}

	w.mu.Lock()
	w.ring = append(w.ring, hit)
	if len(w.ring) > ringSize {
		// Copy into a fresh slice (not just re-slice) so the trimmed
		// prefix's backing array is released rather than the buffer's
		// capacity growing unbounded over the watcher's lifetime.
		trimmed := make([]Hit, ringSize)
		copy(trimmed, w.ring[len(w.ring)-ringSize:])
		w.ring = trimmed
	}
	w.mu.Unlock()

	w.publish(hit)
}

// Recent returns up to the n most recent Hits, oldest first / newest last.
func (w *Watcher) Recent(n int) []Hit {
	w.mu.Lock()
	defer w.mu.Unlock()
	if n <= 0 || n > len(w.ring) {
		n = len(w.ring)
	}
	out := make([]Hit, n)
	copy(out, w.ring[len(w.ring)-n:])
	return out
}

// Subscribe registers a new subscriber and returns a channel that receives
// every subsequently classified Hit, plus an unsubscribe func. The channel
// is best-effort: a slow consumer that doesn't drain it may miss hits, but
// never blocks tailing or other subscribers. Callers must call the
// returned func when done to avoid leaking the subscription.
func (w *Watcher) Subscribe() (<-chan Hit, func()) {
	ch := make(chan Hit, 32)

	w.subsMu.Lock()
	w.subs[ch] = struct{}{}
	w.subsMu.Unlock()

	unsub := func() {
		w.subsMu.Lock()
		delete(w.subs, ch)
		w.subsMu.Unlock()
	}
	return ch, unsub
}

func (w *Watcher) publish(hit Hit) {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	for ch := range w.subs {
		select {
		case ch <- hit:
		default:
			// Slow consumer — drop this hit rather than block tailing or
			// other subscribers.
		}
	}
}

// classify matches line against the signature table (first match wins); if
// none match, an error-ish line (matching an (?i)erro|warn|crit|fatal level
// word — "erro" rather than "error" so lighthouse-pulse's abbreviated ERRO
// tag is caught too) still produces an unclassified Hit (Signature ""),
// severity taken from the level word. A benign line with neither yields
// ok=false — no Hit at all.
func classify(unit, line string, now time.Time) (Hit, bool) {
	for _, sig := range signatures {
		if sig.pattern.MatchString(line) {
			if sig.requireErrLevel && !errLevelPattern.MatchString(line) {
				continue
			}
			return Hit{
				Unit:      unit,
				Line:      line,
				At:        now,
				Signature: sig.name,
				Severity:  sig.severity,
				Explain:   sig.explain,
				LearnURL:  sig.learnURL,
			}, true
		}
	}

	if sev, ok := levelSeverity(line); ok {
		return Hit{
			Unit:     unit,
			Line:     line,
			At:       now,
			Severity: sev,
		}, true
	}

	return Hit{}, false
}

// shQuote single-quotes s for safe interpolation into a `sh -c` command
// string, escaping any embedded single quotes. Unit names are a fixed,
// caller-controlled catalog (not untrusted input), but this keeps the
// command construction consistent with internal/monitor's approach.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
