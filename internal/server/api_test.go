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
