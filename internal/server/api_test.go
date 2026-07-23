package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/valve-node/internal/ai"
	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/config"
	"github.com/valve-tech/valve-node/internal/executor"
)

// ---------------------------------------------------------------------
// test doubles
// ---------------------------------------------------------------------

// autoSucceedExecutor is a fake executor.Executor for this package's API
// tests: uname/df are scripted so setup's preflight step passes, and every
// other command reports a trivial success — enough to exercise the setup
// engine's event plumbing end to end without needing to fake every client
// binary's install/verify probe individually (that's setup's own package
// coverage).
type autoSucceedExecutor struct {
	mu    sync.Mutex
	calls []string
}

func (f *autoSucceedExecutor) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, cmd)
	f.mu.Unlock()
	switch {
	case strings.Contains(cmd, "uname"):
		return executor.Result{Stdout: "Linux\n", ExitCode: 0}, nil
	case strings.Contains(cmd, "df -B1"):
		return executor.Result{Stdout: "9999999999999\n", ExitCode: 0}, nil
	default:
		return executor.Result{ExitCode: 0}, nil
	}
}

func (f *autoSucceedExecutor) WriteFile(_ context.Context, _ string, _ []byte, _ fs.FileMode) error {
	return nil
}

func (f *autoSucceedExecutor) ReadFile(_ context.Context, _ string) ([]byte, error) { return nil, nil }

func (f *autoSucceedExecutor) Close() error { return nil }

// blockingExecutor is a fake executor.Executor for the delete-during-setup
// test: preflight's own probes (uname/df/ss) succeed immediately so
// setup.RunAll gets past the preflight step, then the next command it runs
// (toolchain's git probe) blocks until ctx is canceled — simulating a setup
// step that is still genuinely in flight when a DELETE arrives.
type blockingExecutor struct {
	mu       sync.Mutex
	ranAfter bool // a blocking Run call was actually reached
	closed   bool
	ctxErr   error    // ctx.Err() observed by the blocked Run call
	events   []string // ordering trace: "run-returned", then "closed"
}

func (b *blockingExecutor) Run(ctx context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	switch {
	case strings.Contains(cmd, "uname"):
		return executor.Result{Stdout: "Linux\n", ExitCode: 0}, nil
	case strings.Contains(cmd, "df -B1"):
		return executor.Result{Stdout: "9999999999999\n", ExitCode: 0}, nil
	case strings.Contains(cmd, "ss -ltn"):
		return executor.Result{ExitCode: 0}, nil
	}

	b.mu.Lock()
	b.ranAfter = true
	b.mu.Unlock()

	<-ctx.Done()

	b.mu.Lock()
	b.ctxErr = ctx.Err()
	b.events = append(b.events, "run-returned")
	b.mu.Unlock()

	return executor.Result{}, ctx.Err()
}

func (b *blockingExecutor) WriteFile(_ context.Context, _ string, _ []byte, _ fs.FileMode) error {
	return nil
}

func (b *blockingExecutor) ReadFile(_ context.Context, _ string) ([]byte, error) { return nil, nil }

func (b *blockingExecutor) Close() error {
	b.mu.Lock()
	b.closed = true
	b.events = append(b.events, "closed")
	b.mu.Unlock()
	return nil
}

func (b *blockingExecutor) snapshot() (ranAfter, closed bool, ctxErr error, events []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ranAfter, b.closed, b.ctxErr, append([]string(nil), b.events...)
}

// fakeAIProvider is a stub ai.Provider for the explain route.
type fakeAIProvider struct {
	id      string
	text    string
	err     error
	lastReq ai.ExplainRequest
}

func (f *fakeAIProvider) Name() string { return f.id }

func (f *fakeAIProvider) Explain(_ context.Context, req ai.ExplainRequest) (string, error) {
	f.lastReq = req
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

// apiTestServer bundles a running httptest.Server with the token that
// authorizes it and the underlying *Server, wired with fake executor/AI
// factories and an isolated HOME (so internal/config reads/writes a temp
// dir, never the real user's ~/.valve-node).
type apiTestServer struct {
	ts     *httptest.Server
	token  string
	fakeAI *fakeAIProvider
	home   string
}

func newAPITestServer(t *testing.T) *apiTestServer {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	token := NewSessionToken()
	fake := &fakeAIProvider{text: "canned explanation"}

	s := New(Config{
		Token: token,
		UI: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
		},
		NewExecutor: func(config.Target) (executor.Executor, error) {
			return &autoSucceedExecutor{}, nil
		},
		NewAIProvider: func(id, _, _ string) (ai.Provider, error) {
			fake.id = id
			return fake, nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	return &apiTestServer{ts: ts, token: token, fakeAI: fake, home: home}
}

func (a *apiTestServer) do(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	return a.doRaw(t, method, path, jsonBody(t, body), true)
}

func (a *apiTestServer) doNoAuth(t *testing.T, method, path string) *http.Response {
	t.Helper()
	return a.doRaw(t, method, path, nil, false)
}

func (a *apiTestServer) doRaw(t *testing.T, method, path string, body io.Reader, auth bool) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, a.ts.URL+path, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return res
}

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return bytes.NewReader(b)
}

func decodeJSON[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer res.Body.Close()
	var out T
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// ---------------------------------------------------------------------
// auth gate
// ---------------------------------------------------------------------

func TestEveryAPIRouteRequiresToken(t *testing.T) {
	a := newAPITestServer(t)

	routes := []struct {
		method, path string
	}{
		{"GET", "/api/catalog"},
		{"GET", "/api/targets"},
		{"POST", "/api/targets"},
		{"DELETE", "/api/targets/x"},
		{"POST", "/api/targets/x/setup"},
		{"GET", "/api/targets/x/setup/stream"},
		{"GET", "/api/targets/x/monitor/stream"},
		{"GET", "/api/targets/x/logs"},
		{"GET", "/api/targets/x/logs/stream"},
		{"POST", "/api/targets/x/explain"},
		{"GET", "/api/settings"},
		{"PUT", "/api/settings"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			res := a.doNoAuth(t, rt.method, rt.path)
			defer res.Body.Close()
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s without token: got %d, want 401", rt.method, rt.path, res.StatusCode)
			}
		})
	}
}

// ---------------------------------------------------------------------
// catalog
// ---------------------------------------------------------------------

func TestCatalogReturnsNetworksAndClients(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "GET", "/api/catalog", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var body struct {
		Networks []catalog.Network `json:"networks"`
		Clients  []struct {
			ID string `json:"id"`
		} `json:"clients"`
	}
	body = decodeJSON[struct {
		Networks []catalog.Network `json:"networks"`
		Clients  []struct {
			ID string `json:"id"`
		} `json:"clients"`
	}](t, res)

	if len(body.Networks) == 0 {
		t.Error("networks is empty")
	}
	if len(body.Clients) == 0 {
		t.Error("clients is empty")
	}
}

// ---------------------------------------------------------------------
// target CRUD
// ---------------------------------------------------------------------

func TestTargetCRUDRoundTripsAndPersists(t *testing.T) {
	a := newAPITestServer(t)

	// Empty at first.
	res := a.do(t, "GET", "/api/targets", nil)
	var listed []config.Target
	listed = decodeJSON[[]config.Target](t, res)
	if len(listed) != 0 {
		t.Fatalf("initial targets = %+v, want empty", listed)
	}

	// Add a local target.
	res = a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("POST /api/targets status = %d, want 201, body=%s", res.StatusCode, body)
	}
	added := decodeJSON[config.Target](t, res)
	if added.ID != "local" || added.Mode != "local" {
		t.Fatalf("added target = %+v, want id=local mode=local", added)
	}

	// It shows up in the list.
	res = a.do(t, "GET", "/api/targets", nil)
	listed = decodeJSON[[]config.Target](t, res)
	if len(listed) != 1 || listed[0].ID != "local" {
		t.Fatalf("targets after add = %+v, want [local]", listed)
	}

	// It was actually persisted to disk (not just in-memory).
	t.Setenv("HOME", a.home) // no-op, just documents the shared HOME
	onDisk, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(onDisk.Targets) != 1 || onDisk.Targets[0].ID != "local" {
		t.Fatalf("on-disk targets = %+v, want [local]", onDisk.Targets)
	}

	// Adding the same id again conflicts.
	res = a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate add status = %d, want 409", res.StatusCode)
	}

	// Delete it.
	res = a.do(t, "DELETE", "/api/targets/local", nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", res.StatusCode)
	}

	res = a.do(t, "GET", "/api/targets", nil)
	listed = decodeJSON[[]config.Target](t, res)
	if len(listed) != 0 {
		t.Fatalf("targets after delete = %+v, want empty", listed)
	}

	// Deleting again 404s.
	res = a.do(t, "DELETE", "/api/targets/local", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing target status = %d, want 404", res.StatusCode)
	}
}

// TestHandleAddTargetExecWriteIsRaceFree exercises handleAddTarget's final
// entry.exec write concurrently with getExecutor (which every other path —
// setup/monitor/logs — goes through under entry.mu). This is a white-box
// test (same package) precisely so the concurrent reader can call
// s.getExecutor directly rather than needing to round-trip through a route
// that first requires the target to exist in the on-disk config: both
// handleAddTarget and s.getExecutor share the same *targetEntry the moment
// registry.get(id) is called, and config.Load/Save's real file I/O inside
// handleAddTarget gives the concurrent loop a real (not just adjacent-
// instruction) window to overlap the unguarded write in. Run with
// `go test -race`: before the fix (a bare `entry.exec = ex` write) this
// reliably reports a DATA RACE; after routing through the locked setter,
// it's clean.
func TestHandleAddTargetExecWriteIsRaceFree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s := New(Config{
		Token: NewSessionToken(),
		NewExecutor: func(config.Target) (executor.Executor, error) {
			return &autoSucceedExecutor{}, nil
		},
	})

	target := config.Target{ID: "race-target", Mode: "local"}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			// getExecutor locks entry.mu around every read/write of
			// entry.exec — the same field handleAddTarget's bug wrote to
			// unlocked.
			_, _ = s.getExecutor(target)
		}
	}()

	body, err := json.Marshal(target)
	if err != nil {
		t.Fatalf("marshal target: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleAddTarget(w, req)

	close(stop)
	wg.Wait()

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", w.Code, w.Body.String())
	}
}

func TestAddSSHTargetRequiresConnectionFields(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "POST", "/api/targets", config.Target{ID: "box1", Mode: "ssh"})
	if res.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 400, body=%s", res.StatusCode, body)
	}
}

func TestAddSSHTargetDialsAndDefaultsHostKeyFile(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "POST", "/api/targets", config.Target{
		ID:   "box1",
		Mode: "ssh",
		SSH: &executor.SSHConfig{
			Host:    "10.0.0.5",
			User:    "root",
			KeyPath: "/home/me/.ssh/id_ed25519",
		},
	})
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 201, body=%s", res.StatusCode, body)
	}
	added := decodeJSON[config.Target](t, res)
	if added.SSH == nil || added.SSH.HostKeyFile == "" {
		t.Fatalf("added.SSH = %+v, want a defaulted HostKeyFile", added.SSH)
	}
}

// ---------------------------------------------------------------------
// setup kickoff + SSE replay
// ---------------------------------------------------------------------

func TestSetupKickoffReturns202AndStreamReplaysEvents(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()

	wire := catalog.WireConfig{
		ChainID:  369,
		ExecID:   "reth",
		BeaconID: "lighthouse-pulse",
		DataDir:  "/var/lib/valve-node/369",
	}
	res = a.do(t, "POST", "/api/targets/local/setup", wire)
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("setup kickoff status = %d, want 202, body=%s", res.StatusCode, body)
	}
	res.Body.Close()

	// Give the background RunAll goroutine a moment to emit at least the
	// preflight step's completion before we connect the stream, so this
	// exercises the *replay* path (buffered events sent to a subscriber
	// that connects after they happened), not just live delivery.
	deadline := time.Now().Add(3 * time.Second)
	for {
		res := a.do(t, "GET", "/api/targets/local/setup/stream", nil)
		events := readSSEEventsFor(t, res, 500*time.Millisecond)
		if len(events) > 0 {
			var ev struct {
				StepID string `json:"stepId"`
			}
			if err := json.Unmarshal([]byte(events[0]), &ev); err != nil {
				t.Fatalf("unmarshal event %q: %v", events[0], err)
			}
			if ev.StepID == "" {
				t.Fatalf("event %q has no stepId", events[0])
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for any setup event to appear on the stream")
		}
	}
}

func TestSetupKickoffOnUnknownTargetIs404(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "POST", "/api/targets/nope/setup", catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

// TestDeleteTargetCancelsInFlightSetupBeforeClosingExecutor is the
// use-after-close regression test: it kicks off a real setup run against a
// blockingExecutor, waits for the run to genuinely be blocked mid-step,
// then DELETEs the target. Before the fix, registry.remove closed the
// executor immediately, out from under the still-running setup goroutine.
// After the fix, remove cancels the run's context and waits for the
// goroutine to finish before closing — so the trace must show
// "run-returned" strictly before "closed", the blocked Run call must have
// observed context.Canceled, and DELETE must still return promptly (bounded
// by setupCancelWait, not hung forever).
func TestDeleteTargetCancelsInFlightSetupBeforeClosingExecutor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	fake := &blockingExecutor{}
	token := NewSessionToken()
	s := New(Config{
		Token: token,
		UI: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
		},
		NewExecutor: func(config.Target) (executor.Executor, error) {
			return fake, nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	doReq := func(method, path string, body any) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, jsonBody(t, body))
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return res
	}

	res := doReq("POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()

	res = doReq("POST", "/api/targets/local/setup", catalog.WireConfig{
		ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node/369",
	})
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("setup kickoff status = %d, want 202, body=%s", res.StatusCode, body)
	}
	res.Body.Close()

	// Wait until the setup goroutine has actually reached the blocking
	// step (past preflight) — otherwise the delete below wouldn't be
	// racing a live setup step at all.
	deadline := time.Now().Add(3 * time.Second)
	for {
		if ranAfter, _, _, _ := fake.snapshot(); ranAfter {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for setup to reach the blocking step")
		}
		time.Sleep(5 * time.Millisecond)
	}

	deleteDone := make(chan struct{})
	go func() {
		defer close(deleteDone)
		res := doReq("DELETE", "/api/targets/local", nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusNoContent {
			t.Errorf("DELETE status = %d, want 204", res.StatusCode)
		}
	}()

	select {
	case <-deleteDone:
	case <-time.After(setupCancelWait + 5*time.Second):
		t.Fatal("DELETE did not return — likely blocked on exec.Close() waiting for the in-flight setup step")
	}

	ranAfter, closed, ctxErr, events := fake.snapshot()
	if !ranAfter {
		t.Fatal("blocking step was never reached")
	}
	if ctxErr != context.Canceled {
		t.Errorf("blocked Run's observed ctx.Err() = %v, want context.Canceled", ctxErr)
	}
	if !closed {
		t.Fatal("executor was never closed")
	}
	// toolchainStep's Verify ("git --version") and — since a Verify
	// pre-check failure with a Run set falls through to Run too — its Run
	// ("command -v git") both hit the blocking branch, so more than one
	// "run-returned" is expected; what matters is that every one of them
	// precedes "closed", never the other way around.
	if len(events) < 2 {
		t.Fatalf("event trace = %+v, want at least one run-returned followed by closed", events)
	}
	if last := events[len(events)-1]; last != "closed" {
		t.Fatalf("event trace = %+v, want closed last (executor must be closed only after the setup goroutine finished)", events)
	}
	for _, ev := range events[:len(events)-1] {
		if ev != "run-returned" {
			t.Fatalf("event trace = %+v, want only run-returned entries before the trailing closed", events)
		}
	}
}

// readSSEEventsFor reads "data: ..." lines from an SSE response for up to
// window, then closes the response and returns whatever payloads it saw.
func readSSEEventsFor(t *testing.T, res *http.Response, window time.Duration) []string {
	t.Helper()
	defer res.Body.Close()

	type line struct {
		s   string
		err error
	}
	lines := make(chan line, 64)
	go func() {
		r := bufio.NewReader(res.Body)
		for {
			s, err := r.ReadString('\n')
			if s != "" {
				lines <- line{s: s}
			}
			if err != nil {
				return
			}
		}
	}()

	var out []string
	timer := time.NewTimer(window)
	defer timer.Stop()
	for {
		select {
		case l := <-lines:
			s := strings.TrimSpace(l.s)
			if payload, ok := strings.CutPrefix(s, "data: "); ok {
				out = append(out, payload)
			}
		case <-timer.C:
			return out
		}
	}
}

// ---------------------------------------------------------------------
// monitor stream (per-target)
// ---------------------------------------------------------------------

func TestTargetMonitorStreamRequiresCompletedSetup(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()

	res = a.do(t, "GET", "/api/targets/local/monitor/stream", nil)
	if res.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 409, body=%s", res.StatusCode, body)
	}
}

func TestTargetMonitorStreamDeliversSnapshotAfterSetup(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()

	res = a.do(t, "POST", "/api/targets/local/setup", catalog.WireConfig{
		ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node/369",
	})
	res.Body.Close()

	req, _ := http.NewRequest("GET", a.ts.URL+"/api/targets/local/monitor/stream", nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 200, body=%s", res.StatusCode, body)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	events := readSSEEventsFor(t, res, 500*time.Millisecond)
	if len(events) == 0 {
		t.Fatal("no snapshot events received")
	}
}

// ---------------------------------------------------------------------
// logs
// ---------------------------------------------------------------------

func TestLogsRequiresCompletedSetup(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()

	res = a.do(t, "GET", "/api/targets/local/logs?n=200", nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
}

func TestLogsAfterSetupReturnsJSONArray(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()
	res = a.do(t, "POST", "/api/targets/local/setup", catalog.WireConfig{
		ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node/369",
	})
	res.Body.Close()

	res = a.do(t, "GET", "/api/targets/local/logs?n=200", nil)
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 200, body=%s", res.StatusCode, body)
	}
	var hits []map[string]any
	hits = decodeJSON[[]map[string]any](t, res)
	_ = hits // an empty array is a perfectly valid response; just must decode
}

// ---------------------------------------------------------------------
// explain
// ---------------------------------------------------------------------

func TestExplainWithNoProviderIs409(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()

	res = a.do(t, "POST", "/api/targets/local/explain", map[string]any{})
	if res.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 409, body=%s", res.StatusCode, body)
	}
}

// TestExplainOnUnknownTargetIs404EvenWithoutProvider pins the contract that
// handleExplain checks target existence before checking whether an AI
// provider is configured: an unknown target must 404 regardless of
// settings state, not 409 just because no provider happens to be set up
// yet either.
func TestExplainOnUnknownTargetIs404EvenWithoutProvider(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "POST", "/api/targets/nope/explain", map[string]any{})
	if res.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 404, body=%s", res.StatusCode, body)
	}
}

func TestExplainWithProviderReturnsTextAndSentExcerpt(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()
	res = a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": "gemini", "aiKey": "test-key"})
	res.Body.Close()

	res = a.do(t, "POST", "/api/targets/local/explain", map[string]any{
		"lines": []string{"FATAL something broke"},
	})
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, want 200, body=%s", res.StatusCode, body)
	}
	var body struct {
		Text        string   `json:"text"`
		SentExcerpt []string `json:"sentExcerpt"`
	}
	body = decodeJSON[struct {
		Text        string   `json:"text"`
		SentExcerpt []string `json:"sentExcerpt"`
	}](t, res)
	if body.Text != "canned explanation" {
		t.Errorf("text = %q, want the fake provider's canned text", body.Text)
	}
	if len(body.SentExcerpt) != 1 || body.SentExcerpt[0] != "FATAL something broke" {
		t.Errorf("sentExcerpt = %+v, want the line we sent", body.SentExcerpt)
	}
	if a.fakeAI.id != "gemini" {
		t.Errorf("provider factory was called with id = %q, want gemini", a.fakeAI.id)
	}
}

// ---------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------

func TestSettingsPutStoresKeyGetMasksIt(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "GET", "/api/settings", nil)
	initial := decodeJSON[map[string]any](t, res)
	if initial["aiKeySet"] != false {
		t.Fatalf("initial aiKeySet = %v, want false", initial["aiKeySet"])
	}

	res = a.do(t, "PUT", "/api/settings", map[string]any{
		"aiProvider": "groq",
		"aiKey":      "super-secret",
	})
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("PUT status = %d, want 200, body=%s", res.StatusCode, body)
	}
	putResp := decodeJSON[map[string]any](t, res)
	if putResp["aiKeySet"] != true {
		t.Fatalf("PUT response aiKeySet = %v, want true", putResp["aiKeySet"])
	}
	if _, leaked := putResp["aiKey"]; leaked {
		t.Fatalf("PUT response leaked the raw key: %+v", putResp)
	}

	res = a.do(t, "GET", "/api/settings", nil)
	got := decodeJSON[map[string]any](t, res)
	if got["aiProvider"] != "groq" {
		t.Errorf("aiProvider = %v, want groq", got["aiProvider"])
	}
	if got["aiKeySet"] != true {
		t.Errorf("aiKeySet = %v, want true", got["aiKeySet"])
	}
	if _, leaked := got["aiKey"]; leaked {
		t.Fatalf("GET response leaked the raw key: %+v", got)
	}

	// The key really was persisted to disk (not just held in memory).
	onDisk, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if onDisk.AIKey != "super-secret" {
		t.Errorf("on-disk AIKey = %q, want super-secret", onDisk.AIKey)
	}
}

// TestSettingsPutMaskedFieldSemantics pins the settingsRequest pointer-field
// contract: omitting aiKey on a PUT must leave a previously stored key
// untouched (a client re-PUTting the masked response of a prior GET must
// not blow the key away), while an explicit empty string must clear it.
func TestSettingsPutMaskedFieldSemantics(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": "groq", "aiKey": "k1"})
	res.Body.Close()

	res = a.do(t, "GET", "/api/settings", nil)
	got := decodeJSON[map[string]any](t, res)
	if got["aiKeySet"] != true {
		t.Fatalf("aiKeySet after PUT aiKey=k1 = %v, want true", got["aiKeySet"])
	}

	// Switch provider without mentioning aiKey at all — the stored key
	// must survive untouched.
	res = a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": "gemini"})
	res.Body.Close()

	res = a.do(t, "GET", "/api/settings", nil)
	got = decodeJSON[map[string]any](t, res)
	if got["aiProvider"] != "gemini" {
		t.Fatalf("aiProvider = %v, want gemini", got["aiProvider"])
	}
	if got["aiKeySet"] != true {
		t.Fatalf("aiKeySet after omitted-aiKey PUT = %v, want still true (key preserved)", got["aiKeySet"])
	}
	onDisk, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if onDisk.AIKey != "k1" {
		t.Fatalf("on-disk AIKey after omitted-aiKey PUT = %q, want unchanged %q", onDisk.AIKey, "k1")
	}

	// An explicit empty aiKey clears it.
	res = a.do(t, "PUT", "/api/settings", map[string]any{"aiKey": ""})
	res.Body.Close()

	res = a.do(t, "GET", "/api/settings", nil)
	got = decodeJSON[map[string]any](t, res)
	if got["aiKeySet"] != false {
		t.Fatalf("aiKeySet after explicit empty aiKey PUT = %v, want false", got["aiKeySet"])
	}
	onDisk, err = config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if onDisk.AIKey != "" {
		t.Fatalf("on-disk AIKey after explicit empty aiKey PUT = %q, want empty", onDisk.AIKey)
	}
}
