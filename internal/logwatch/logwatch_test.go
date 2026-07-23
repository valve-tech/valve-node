package logwatch

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node/internal/executor"
)

// ---------------------------------------------------------------------
// fakeExecutor — mirrors internal/monitor and internal/setup's test
// double: a scripted map[string]executor.Result keyed by command substring
// (longest match wins), with every Run call recorded in order. When
// opts.Stream is set, the scripted Stdout is replayed line-by-line through
// it — this is how tailing `journalctl -u <unit> -f` is simulated.
// ---------------------------------------------------------------------

type fakeExecutor struct {
	mu      sync.Mutex
	scripts map[string]executor.Result
	calls   []string
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{scripts: map[string]executor.Result{}}
}

func (f *fakeExecutor) script(substr string, res executor.Result) *fakeExecutor {
	f.scripts[substr] = res
	return f
}

func (f *fakeExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, cmd)
	f.mu.Unlock()

	var matchKey string
	var res executor.Result
	f.mu.Lock()
	for k, v := range f.scripts {
		if strings.Contains(cmd, k) && len(k) > len(matchKey) {
			matchKey, res = k, v
		}
	}
	f.mu.Unlock()

	if matchKey != "" && opts != nil && opts.Stream != nil {
		for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n") {
			if line != "" {
				opts.Stream(line)
			}
		}
	}

	// Real `journalctl -u <unit> -f` blocks until the context is canceled
	// (or the transport drops); block here too so it doesn't look like Run
	// returned on its own once the scripted backlog has been replayed —
	// that would spuriously trip Watcher.tail's retry loop in unrelated
	// tests.
	<-ctx.Done()
	return res, ctx.Err()
}

func (f *fakeExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	return nil
}
func (f *fakeExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (f *fakeExecutor) Close() error                                              { return nil }

func (f *fakeExecutor) callLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

// ---------------------------------------------------------------------
// classify: fixture table — one real-shaped line per signature, three
// benign lines (no hit), two unclassified-error lines (level word, no
// signature).
// ---------------------------------------------------------------------

func TestClassify(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		wantHit   bool
		wantSig   string
		wantSev   string
		wantExpl  bool // Explain non-empty
		wantLearn bool // LearnURL non-empty
	}{
		{
			name:    "beacon stalled sync state",
			line:    "INFO Sync state updated state=Stalled distance=120",
			wantHit: true, wantSig: "beacon-stalled", wantSev: "critical", wantExpl: true,
		},
		{
			name:    "engine auth jwt/401",
			line:    `ERRO Failed to connect to execution client err="401 Unauthorized: invalid jwt"`,
			wantHit: true, wantSig: "engine-auth", wantSev: "critical", wantExpl: true,
		},
		{
			name:    "checkpoint sync failure",
			line:    "ERRO checkpoint sync failed for url=https://checkpoint.example retrying",
			wantHit: true, wantSig: "checkpoint-sync-failed", wantSev: "error", wantExpl: true,
		},
		{
			name:    "low peer count",
			line:    "WARN Low peer count peers=2",
			wantHit: true, wantSig: "low-peer-count", wantSev: "warn", wantExpl: true,
		},
		{
			name:    "disk full",
			line:    "FATAL write /var/lib/reth/db/mdbx.dat: no space left on device",
			wantHit: true, wantSig: "disk-full", wantSev: "critical", wantExpl: true,
		},
		{
			name:    "database corrupt",
			line:    "FATAL Fatal error: database is corrupt, please resync from a snapshot",
			wantHit: true, wantSig: "database-corrupt", wantSev: "critical", wantExpl: true,
		},
		{
			name:    "oom killed",
			line:    "kernel: Out of memory: Killed process 5821 (reth) score 900",
			wantHit: true, wantSig: "oom-killed", wantSev: "critical", wantExpl: true,
		},
		{
			name:    "port in use",
			line:    "FATAL Failed to start server: listen tcp 127.0.0.1:8551: bind: address already in use",
			wantHit: true, wantSig: "port-in-use", wantSev: "critical", wantExpl: true,
		},
		// benign — no hit at all.
		{
			name:    "benign import",
			line:    "INFO Imported new chain segment number=123456 hash=0xabc elapsed=12.3ms",
			wantHit: false,
		},
		{
			name:    "benign peer discovery",
			line:    "INFO Looking for peers, discovered 5 candidates",
			wantHit: false,
		},
		{
			name:    "benign head update",
			line:    "DEBUG Head state updated slot=987654",
			wantHit: false,
		},
		// unclassified — level word present, no signature match.
		{
			name:    "unclassified error",
			line:    "ERROR panic: index out of range [3] with length 3",
			wantHit: true, wantSig: "", wantSev: "error",
		},
		{
			name:    "unclassified warn",
			line:    "WARN unexpected response from peer, dropping connection",
			wantHit: true, wantSig: "", wantSev: "warn",
		},
		{
			name:    "lighthouse abbreviated ERRO level tag",
			line:    "Jul 23 06:00:00 ERRO Head is stuck, prune check failed",
			wantHit: true, wantSig: "", wantSev: "error",
		},
		// engine-auth must not fire on benign lines that merely mention
		// jwt/401/unauthorized without an error-level indicator — same
		// lesson as setup's handshake authErrorLines gate (steps.go),
		// propagated here.
		{
			name:    "benign prysm JWT read line, no engine-auth false positive",
			line:    `level=info msg="Finished reading JWT secret from /x/jwt.hex"`,
			wantHit: false,
		},
		{
			name:    "engine auth with explicit error level still fires",
			line:    `level=error msg="401 Unauthorized invalid JWT"`,
			wantHit: true, wantSig: "engine-auth", wantSev: "critical", wantExpl: true,
		},
		// low-peer-count's "0 peers" match must be word-bounded so it
		// doesn't substring-match "10 peers"/"20 peers".
		{
			name:    "ten peers is not zero peers",
			line:    "INFO Connected to 10 peers",
			wantHit: false,
		},
		{
			name:    "zero peers alone is low-peer-count",
			line:    "WARN 0 peers",
			wantHit: true, wantSig: "low-peer-count", wantSev: "warn", wantExpl: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hit, ok := classify("valve-node-exec.service", tc.line, time.Now())
			if ok != tc.wantHit {
				t.Fatalf("classify(%q) ok = %v, want %v", tc.line, ok, tc.wantHit)
			}
			if !tc.wantHit {
				return
			}
			if hit.Signature != tc.wantSig {
				t.Errorf("Signature = %q, want %q", hit.Signature, tc.wantSig)
			}
			if hit.Severity != tc.wantSev {
				t.Errorf("Severity = %q, want %q", hit.Severity, tc.wantSev)
			}
			if tc.wantExpl && hit.Explain == "" {
				t.Errorf("Explain is empty, want non-empty canned explanation")
			}
			if !tc.wantExpl && hit.Explain != "" {
				t.Errorf("Explain = %q, want empty for unclassified line", hit.Explain)
			}
			if hit.Line != tc.line {
				t.Errorf("Line = %q, want %q", hit.Line, tc.line)
			}
			if hit.Unit != "valve-node-exec.service" {
				t.Errorf("Unit = %q, want valve-node-exec.service", hit.Unit)
			}
			if hit.At.IsZero() {
				t.Errorf("At is zero, want set")
			}
		})
	}
}

func TestClassifyLearnURLOnlyWhenSignatureHasOne(t *testing.T) {
	// engine-auth has a learn page (JWT setup is covered by the RPC
	// node-setup guide); database-corrupt does not.
	hit, ok := classify("u", `401 Unauthorized invalid jwt`, time.Now())
	if !ok {
		t.Fatal("expected a hit")
	}
	if hit.LearnURL == "" {
		t.Errorf("engine-auth: LearnURL empty, want a learn.valve.city/rpc link")
	}
	if !strings.HasPrefix(hit.LearnURL, "https://learn.valve.city/rpc") {
		t.Errorf("LearnURL = %q, want it based on https://learn.valve.city/rpc", hit.LearnURL)
	}

	hit2, ok := classify("u", "database is corrupt", time.Now())
	if !ok {
		t.Fatal("expected a hit")
	}
	if hit2.LearnURL != "" {
		t.Errorf("database-corrupt: LearnURL = %q, want empty (no learn page)", hit2.LearnURL)
	}
}

// ---------------------------------------------------------------------
// Watcher: tail command shape
// ---------------------------------------------------------------------

func TestStartRunsJournalctlFollowPerUnit(t *testing.T) {
	fe := newFakeExecutor()
	w := New(fe, []string{"valve-node-exec.service", "valve-node-beacon.service"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	deadline := time.After(2 * time.Second)
	for {
		calls := fe.callLog()
		if len(calls) >= 2 {
			foundExec, foundBeacon := false, false
			for _, c := range calls {
				if strings.Contains(c, "journalctl") && strings.Contains(c, "-f") && strings.Contains(c, "valve-node-exec.service") {
					foundExec = true
				}
				if strings.Contains(c, "journalctl") && strings.Contains(c, "-f") && strings.Contains(c, "valve-node-beacon.service") {
					foundBeacon = true
				}
			}
			if foundExec && foundBeacon {
				return
			}
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for journalctl -f calls, got: %v", calls)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// ---------------------------------------------------------------------
// Watcher: classification end-to-end via Stream, ring buffer, Subscribe
// ---------------------------------------------------------------------

func TestWatcherClassifiesStreamedLinesIntoRingBuffer(t *testing.T) {
	fe := newFakeExecutor().script("valve-node-exec.service", executor.Result{
		Stdout: "INFO Imported new chain segment number=1\n" +
			"WARN Low peer count peers=1\n" +
			"ERROR panic: something broke\n",
	})
	w := New(fe, []string{"valve-node-exec.service"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	waitForRecent(t, w, 2)

	hits := w.Recent(10)
	if len(hits) != 2 {
		t.Fatalf("Recent(10) len = %d, want 2 (benign line produces no hit)", len(hits))
	}
	if hits[0].Signature != "low-peer-count" {
		t.Errorf("hits[0].Signature = %q, want low-peer-count", hits[0].Signature)
	}
	if hits[1].Signature != "" || hits[1].Severity != "error" {
		t.Errorf("hits[1] = %+v, want unclassified error", hits[1])
	}
}

func TestRecentReturnsNewestLast(t *testing.T) {
	fe := newFakeExecutor().script("valve-node-exec.service", executor.Result{
		Stdout: "ERROR one\nERROR two\nERROR three\n",
	})
	w := New(fe, []string{"valve-node-exec.service"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	waitForRecent(t, w, 3)

	hits := w.Recent(10)
	if len(hits) != 3 {
		t.Fatalf("Recent(10) len = %d, want 3", len(hits))
	}
	if hits[0].Line != "ERROR one" || hits[1].Line != "ERROR two" || hits[2].Line != "ERROR three" {
		t.Errorf("Recent order = %v, want one, two, three (newest last)", []string{hits[0].Line, hits[1].Line, hits[2].Line})
	}
}

func TestRecentCapsAtRingSize(t *testing.T) {
	var b strings.Builder
	for i := 0; i < ringSize+50; i++ {
		fmt.Fprintf(&b, "ERROR line %d\n", i)
	}
	fe := newFakeExecutor().script("valve-node-exec.service", executor.Result{Stdout: b.String()})
	w := New(fe, []string{"valve-node-exec.service"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// ring length alone hits ringSize well before all ringSize+50 lines
	// have been streamed (it plateaus at ringSize once eviction starts),
	// so wait for the specific last line to show up instead.
	wantLast := fmt.Sprintf("ERROR line %d", ringSize+49)
	deadline := time.After(2 * time.Second)
	for {
		hits := w.Recent(1)
		if len(hits) == 1 && hits[0].Line == wantLast {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for last line %q to be classified", wantLast)
		case <-time.After(10 * time.Millisecond):
		}
	}

	hits := w.Recent(ringSize + 100)
	if len(hits) != ringSize {
		t.Fatalf("Recent len = %d, want capped at ringSize=%d", len(hits), ringSize)
	}
	// The oldest 50 lines (0..49) should have been evicted; the buffer
	// should now start at line 50 and end at line ringSize+49.
	if hits[0].Line != fmt.Sprintf("ERROR line %d", 50) {
		t.Errorf("hits[0].Line = %q, want %q (oldest entries evicted)", hits[0].Line, fmt.Sprintf("ERROR line %d", 50))
	}
	last := ringSize + 49
	if hits[len(hits)-1].Line != fmt.Sprintf("ERROR line %d", last) {
		t.Errorf("last hit Line = %q, want %q", hits[len(hits)-1].Line, fmt.Sprintf("ERROR line %d", last))
	}
}

func TestSubscribeDeliversHits(t *testing.T) {
	fe := newFakeExecutor().script("valve-node-exec.service", executor.Result{
		Stdout: "ERROR boom\n",
	})
	w := New(fe, []string{"valve-node-exec.service"})

	ch, unsub := w.Subscribe()
	defer unsub()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	select {
	case hit := <-ch:
		if hit.Line != "ERROR boom" {
			t.Errorf("hit.Line = %q, want %q", hit.Line, "ERROR boom")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a subscribed hit")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	w := New(newFakeExecutor(), []string{"valve-node-exec.service"})
	ch, unsub := w.Subscribe()
	unsub()

	select {
	case v, ok := <-ch:
		if ok {
			t.Fatalf("received a hit after unsubscribe: %+v", v)
		}
	case <-time.After(50 * time.Millisecond):
		// No delivery — correct (channel isn't closed, just no longer fed).
	}
}

// ---------------------------------------------------------------------
// Watcher: tail retry-with-backoff when Run returns for a reason other
// than ctx cancellation (SSH transport drop, journald restart).
// ---------------------------------------------------------------------

// retryExec's Run returns immediately (no error) for the first `immediate`
// calls, then blocks until ctx is done on every call after that —
// simulating a transport that drops a few times and then holds a stable
// follow. Used to prove Watcher.tail re-invokes Run rather than giving up
// after the first non-cancel return.
type retryExec struct {
	mu        sync.Mutex
	immediate int
	calls     int
}

func (r *retryExec) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	r.calls++
	n := r.calls
	r.mu.Unlock()

	if n <= r.immediate {
		return executor.Result{}, nil
	}
	<-ctx.Done()
	return executor.Result{}, ctx.Err()
}

func (r *retryExec) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	return nil
}
func (r *retryExec) ReadFile(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (r *retryExec) Close() error                                              { return nil }

func (r *retryExec) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func TestTailRetriesRunAfterNonCancelReturn(t *testing.T) {
	re := &retryExec{immediate: 3}
	w := New(re, []string{"valve-node-exec.service"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Run must be re-invoked (>=2 times) while ctx is still alive, proving
	// a non-cancel return doesn't kill tailing permanently.
	deadline := time.After(6 * time.Second)
	for {
		if re.callCount() >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for Run to be retried, got %d calls", re.callCount())
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()

	// After cancel, the retry loop must stop: the call count should not
	// keep climbing.
	afterCancel := re.callCount()
	time.Sleep(100 * time.Millisecond)
	if got := re.callCount(); got > afterCancel+1 {
		t.Errorf("Run kept being retried after cancel: %d -> %d", afterCancel, got)
	}
}

// waitForRecent polls until Recent(want) has at least `want` entries or
// fails the test after a timeout.
func waitForRecent(t *testing.T, w *Watcher, want int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if len(w.Recent(want)) >= want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d hits, got %d", want, len(w.Recent(want)))
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestClassify_Exported(t *testing.T) {
	now := time.Now()
	hit, ok := Classify("valve-node-exec.service", "WARN low peer count: 0 peers", now)
	if !ok {
		t.Fatal("want a Hit for a low-peer-count line, got ok=false")
	}
	if hit.Signature != "low-peer-count" {
		t.Fatalf("Signature = %q, want %q", hit.Signature, "low-peer-count")
	}
	if hit.Explain == "" {
		t.Fatal("want a non-empty canned Explain for a named signature")
	}
	if hit.Unit != "valve-node-exec.service" || !hit.At.Equal(now) {
		t.Fatalf("Unit/At not carried through: %+v", hit)
	}

	if _, ok := Classify("valve-node-exec.service", "INFO imported new chain segment", now); ok {
		t.Fatal("want ok=false for a benign info line")
	}
}
