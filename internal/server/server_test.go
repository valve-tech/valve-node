package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
	"github.com/valve-tech/valve-node/internal/monitor"
)

func testServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func TestHealthRequiresToken(t *testing.T) {
	ts, token := testServer(t)
	res, _ := http.Get(ts.URL + "/api/health")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", res.StatusCode)
	}
	req, _ := http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("with token: got %d, want 200", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != `{"ok":true}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestTokenQueryParamSetsCookieAndServesUI(t *testing.T) {
	ts, token := testServer(t)
	jar, _ := cookiejarNew()
	client := &http.Client{Jar: jar}
	res, _ := client.Get(ts.URL + "/?token=" + token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ui with token: %d", res.StatusCode)
	}
	// Cookie now authorizes the API without the header.
	res, _ = client.Get(ts.URL + "/api/health")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("api via cookie: %d", res.StatusCode)
	}
}

func TestNewSessionTokenIsRandomHex(t *testing.T) {
	a, b := NewSessionToken(), NewSessionToken()
	if len(a) != 32 || a == b {
		t.Fatalf("tokens: %q %q", a, b)
	}
}

// noopExecutor is a minimal executor.Executor stub: every command
// "succeeds" with empty output, which is enough to drive monitor.Monitor
// end to end for this package's SSE-plumbing test (monitor's own package
// covers probe parsing).
type noopExecutor struct{}

func (noopExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	return executor.Result{ExitCode: 0}, nil
}

func (noopExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	return nil
}

func (noopExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) { return nil, nil }

func (noopExecutor) Close() error { return nil }

func TestMonitorStreamRequiresToken(t *testing.T) {
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	res, _ := http.Get(ts.URL + "/api/monitor/stream")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", res.StatusCode)
	}
}

func TestMonitorStreamWithoutMonitorConfigured(t *testing.T) {
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/monitor/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503 when no Monitor is configured", res.StatusCode)
	}
}

func TestMonitorStreamDeliversJSONSnapshot(t *testing.T) {
	m := monitor.New(monitor.Config{
		Exec:     noopExecutor{},
		Wire:     catalog.WireConfig{ChainID: 369, DataDir: "/var/lib/valve-node/369"},
		Interval: 10 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	token := NewSessionToken()
	s := New(Config{Token: token, Monitor: m, UI: fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	reqCtx, reqCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer reqCancel()
	req, err := http.NewRequestWithContext(reqCtx, "GET", ts.URL+"/api/monitor/stream", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	reader := bufio.NewReader(res.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read first SSE line: %v", err)
	}
	line = strings.TrimSpace(line)
	payload, ok := strings.CutPrefix(line, "data: ")
	if !ok {
		t.Fatalf("first SSE line = %q, want a %q-prefixed event", line, "data: ")
	}

	var snap monitor.Snapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		t.Fatalf("unmarshal snapshot event %q: %v", payload, err)
	}
}
