// This file implements the JSON/SSE API: target CRUD, the setup wizard
// kickoff + progress stream, per-target monitor/log streams, AI log
// explanations, and provider/key settings. Every route here is mounted
// under the authMiddleware in server.go's Handler, so nothing in this file
// needs to re-check the session token.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/valve-tech/valve-node/internal/ai"
	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/config"
	"github.com/valve-tech/valve-node/internal/executor"
	"github.com/valve-tech/valve-node/internal/logwatch"
	"github.com/valve-tech/valve-node/internal/monitor"
	"github.com/valve-tech/valve-node/internal/ops"
	"github.com/valve-tech/valve-node/internal/setup"
)

// logUnits is the fixed set of journald units every target's logwatch
// Watcher tails, matching internal/monitor's unit-name constants.
var logUnits = []string{"valve-node-exec.service", "valve-node-beacon.service"}

// defaultRecentLogs is used when GET .../logs is called without ?n=.
const defaultRecentLogs = 200

// ---------------------------------------------------------------------
// registry: per-target runtime state (executor, monitor, log watcher,
// setup run), created lazily on first use and kept for the life of the
// Server process (or until the target is deleted).
// ---------------------------------------------------------------------

type registry struct {
	mu      sync.Mutex
	entries map[string]*targetEntry
}

func newRegistry() *registry {
	return &registry{entries: map[string]*targetEntry{}}
}

// get returns the entry for id, creating an empty one if this is the first
// time id has been seen.
func (r *registry) get(id string) *targetEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[id]
	if !ok {
		e = &targetEntry{}
		r.entries[id] = e
	}
	return e
}

// setupCancelWait bounds how long registry.remove waits for an in-flight
// setup run to observe cancellation and stop touching the target's
// executor before remove closes that executor out from under it.
const setupCancelWait = 5 * time.Second

// remove evicts id's entry, stopping its monitor/watcher goroutines,
// canceling and waiting (bounded) for any in-flight setup run, and only
// then closing its cached executor — closing the executor before an
// in-flight setup.RunAll goroutine has actually stopped using it would be a
// use-after-close race.
func (r *registry) remove(id string) {
	r.mu.Lock()
	e, ok := r.entries[id]
	delete(r.entries, id)
	r.mu.Unlock()
	if !ok {
		return
	}

	e.mu.Lock()
	if e.monStop != nil {
		e.monStop()
	}
	if e.watchStop != nil {
		e.watchStop()
	}
	run := e.setup
	e.mu.Unlock()

	if run != nil {
		run.cancelAndWait(setupCancelWait)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.exec != nil {
		e.exec.Close()
	}
}

type targetEntry struct {
	mu sync.Mutex

	exec executor.Executor

	mon     *monitor.Monitor
	monStop context.CancelFunc

	watch     *logwatch.Watcher
	watchStop context.CancelFunc

	setup *setupRun

	// Network-diagnostics state, guarded by its own mutex because auto-run
	// goroutines touch it while entry.mu may be held by slow executor
	// dials. See diag.go for the gate semantics.
	diagMu     sync.Mutex
	diagLatest *DiagReport
	diagLast   time.Time
	diagBusy   bool
}

// setExec caches ex as entry's executor under entry.mu. handleAddTarget
// dials an executor before entry.mu is ever taken (it doesn't need the
// entry until this point), so it must go through this locked setter rather
// than writing entry.exec directly — otherwise it races every other path
// (getExecutorLocked, registry.remove) that touches the same field under
// the lock. If a concurrent getExecutorLocked call for the same id already
// cached one first, the redundant dial is closed and the existing one wins.
func (e *targetEntry) setExec(ex executor.Executor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.exec != nil {
		ex.Close()
		return
	}
	e.exec = ex
}

// setupRun tracks one setup.RunAll invocation for a target: every event it
// has emitted so far (so a new SSE subscriber can replay history) plus the
// live subscriber set.
type setupRun struct {
	mu      sync.Mutex
	events  []setup.Event
	subs    map[chan setup.Event]struct{}
	running bool
	err     error

	// cancel and done let registry.remove interrupt an in-flight run and
	// wait (bounded) for its goroutine to actually stop touching the
	// target's executor before Close()ing it — see cancelAndWait. done is
	// closed by the setup goroutine once setup.RunAll has returned.
	cancel context.CancelFunc
	done   chan struct{}
}

func newSetupRun(cancel context.CancelFunc) *setupRun {
	return &setupRun{
		subs:    map[chan setup.Event]struct{}{},
		running: true,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

// cancelAndWait cancels the run's context, then blocks for up to timeout
// waiting for the run's goroutine to finish (signaled by done being
// closed) — so a caller about to Close the run's executor can be sure it's
// no longer in use, without risking an unbounded wait if the run's step
// ignores cancellation somehow.
func (sr *setupRun) cancelAndWait(timeout time.Duration) {
	sr.mu.Lock()
	cancel := sr.cancel
	done := sr.done
	sr.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (sr *setupRun) append(ev setup.Event) {
	sr.mu.Lock()
	sr.events = append(sr.events, ev)
	for ch := range sr.subs {
		select {
		case ch <- ev:
		default:
			// Slow consumer — drop, matching monitor/logwatch's Subscribe
			// contract. The replay-on-connect behavior means a dropped live
			// tick is never permanently lost to a *new* subscriber anyway.
		}
	}
	sr.mu.Unlock()
}

func (sr *setupRun) finish(err error) {
	sr.mu.Lock()
	sr.running = false
	sr.err = err
	sr.mu.Unlock()
}

// subscribe returns every event emitted so far plus a channel that
// receives every subsequent one, registered atomically so no event can be
// missed or duplicated across the snapshot/live-feed boundary.
func (sr *setupRun) subscribe() ([]setup.Event, chan setup.Event, func()) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	snapshot := append([]setup.Event(nil), sr.events...)
	ch := make(chan setup.Event, 32)
	sr.subs[ch] = struct{}{}
	unsub := func() {
		sr.mu.Lock()
		delete(sr.subs, ch)
		sr.mu.Unlock()
	}
	return snapshot, ch, unsub
}

// ---------------------------------------------------------------------
// executor construction
// ---------------------------------------------------------------------

// defaultNewExecutor is Server.newExecutor's default: a real local or SSH
// executor depending on Target.Mode.
func defaultNewExecutor(t config.Target) (executor.Executor, error) {
	switch t.Mode {
	case "local":
		return executor.NewLocal(), nil
	case "ssh":
		if t.SSH == nil {
			return nil, fmt.Errorf("target %q: mode \"ssh\" requires an ssh config", t.ID)
		}
		return executor.NewSSH(*t.SSH)
	default:
		return nil, fmt.Errorf("target %q: unknown mode %q", t.ID, t.Mode)
	}
}

// getExecutor returns t's cached executor, dialing and caching a new one on
// first use.
func (s *Server) getExecutor(t config.Target) (executor.Executor, error) {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return s.getExecutorLocked(entry, t)
}

// getExecutorLocked is getExecutor's body, for callers that already hold
// entry.mu (avoids re-entrant locking).
func (s *Server) getExecutorLocked(entry *targetEntry, t config.Target) (executor.Executor, error) {
	if entry.exec != nil {
		return entry.exec, nil
	}
	ex, err := s.newExecutor(t)
	if err != nil {
		return nil, err
	}
	entry.exec = ex
	return ex, nil
}

// getMonitor returns t's monitor.Monitor, lazily creating and starting one
// (polling forever, until the target is deleted) on first use.
func (s *Server) getMonitor(t config.Target, refRPCBase string) (*monitor.Monitor, error) {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.mon != nil {
		return entry.mon, nil
	}
	ex, err := s.getExecutorLocked(entry, t)
	if err != nil {
		return nil, err
	}
	refRPC := ""
	if refRPCBase != "" {
		refRPC = fmt.Sprintf("%s/evm/%d", refRPCBase, t.Wire.ChainID)
	}
	mon := monitor.New(monitor.Config{Exec: ex, Wire: *t.Wire, RefRPC: refRPC})
	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)
	// Auto-diagnostics trigger: failed connections (inactive service, zero
	// peers) in this monitor's snapshots kick off a background diagnostics
	// run, gated by the per-target cooldown (diag.go).
	go s.watchMonitorForDiag(ctx, t, mon)
	entry.mon = mon
	entry.monStop = cancel
	return mon, nil
}

// getWatcher returns t's logwatch.Watcher, lazily creating and starting one
// (tailing forever, until the target is deleted) on first use.
func (s *Server) getWatcher(t config.Target) (*logwatch.Watcher, error) {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.watch != nil {
		return entry.watch, nil
	}
	ex, err := s.getExecutorLocked(entry, t)
	if err != nil {
		return nil, err
	}
	watch := logwatch.New(ex, logUnits)
	ctx, cancel := context.WithCancel(context.Background())
	watch.Start(ctx)
	// Auto-diagnostics trigger: error/critical journal hits kick off a
	// background diagnostics run, gated by the per-target cooldown
	// (diag.go).
	go s.watchLogsForDiag(ctx, t, watch)
	entry.watch = watch
	entry.watchStop = cancel
	return watch, nil
}

// ---------------------------------------------------------------------
// config helpers
// ---------------------------------------------------------------------

func (s *Server) loadConfig() (config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	return config.Load()
}

// updateConfig loads the config, applies fn, saves it, and returns the
// saved result — all under cfgMu so concurrent API requests can't clobber
// each other's edits.
func (s *Server) updateConfig(fn func(c *config.Config) error) (config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	c, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	if err := fn(&c); err != nil {
		return config.Config{}, err
	}
	if err := c.Save(); err != nil {
		return config.Config{}, err
	}
	return c, nil
}

func findTarget(cfg config.Config, id string) (config.Target, bool) {
	for _, t := range cfg.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return config.Target{}, false
}

// ---------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeSSEEvent marshals v and writes it as one `data: <json>\n\n` SSE
// frame. Marshal failures are dropped silently — there is no way to report
// an error mid-stream that wouldn't also break the stream framing.
func writeSSEEvent(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
}

func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------
// routes
// ---------------------------------------------------------------------

func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/catalog", s.handleCatalog)

	mux.HandleFunc("GET /api/targets", s.handleListTargets)
	mux.HandleFunc("POST /api/targets", s.handleAddTarget)
	mux.HandleFunc("DELETE /api/targets/{id}", s.handleDeleteTarget)

	mux.HandleFunc("POST /api/targets/{id}/setup", s.handleStartSetup)
	mux.HandleFunc("GET /api/targets/{id}/setup/stream", s.handleSetupStream)

	mux.HandleFunc("GET /api/targets/{id}/monitor/stream", s.handleTargetMonitorStream)

	mux.HandleFunc("GET /api/targets/{id}/logs", s.handleLogs)
	mux.HandleFunc("GET /api/targets/{id}/logs/stream", s.handleLogsStream)

	mux.HandleFunc("POST /api/targets/{id}/explain", s.handleExplain)

	// The literal "clear" segment is more specific than the {action}
	// wildcard below it and wins for an exact match — Go 1.22+ ServeMux
	// prefers the more specific pattern regardless of registration order —
	// so these two don't collide.
	mux.HandleFunc("POST /api/targets/{id}/services/{svc}/clear", s.handleServiceClear)
	mux.HandleFunc("POST /api/targets/{id}/services/{svc}/{action}", s.handleServiceAction)
	mux.HandleFunc("GET /api/targets/{id}/du", s.handleDiskUsage)
	mux.HandleFunc("GET /api/targets/{id}/endpoints", s.handleEndpoints)
	mux.HandleFunc("GET /api/targets/{id}/firewall", s.handleFirewall)
	mux.HandleFunc("GET /api/targets/{id}/diagnostics", s.handleDiagnostics)
	mux.HandleFunc("GET /api/targets/{id}/diagnostics/latest", s.handleDiagnosticsLatest)

	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handlePutSettings)
}

// ---------------------------------------------------------------------
// GET /api/catalog
// ---------------------------------------------------------------------

type catalogClient struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Repo       string `json:"repo"`
	PinVersion string `json:"pinVersion"`
	Toolchain  string `json:"toolchain"`
	LearnURL   string `json:"learnUrl"`
}

type catalogResponse struct {
	Networks []catalog.Network `json:"networks"`
	Clients  []catalogClient   `json:"clients"`
}

// handleCatalog returns every known network plus every client referenced by
// any of them. catalog.Client isn't itself JSON-safe (ReleaseURL is a
// func), and catalog exposes no "all clients" listing, so the client set is
// derived from the ids each Network's ExecClients/BeaconClients names.
func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	networks := catalog.Networks()

	seen := map[string]struct{}{}
	for _, n := range networks {
		for _, id := range n.ExecClients {
			seen[id] = struct{}{}
		}
		for _, id := range n.BeaconClients {
			seen[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	clients := make([]catalogClient, 0, len(ids))
	for _, id := range ids {
		c, ok := catalog.ClientByID(id)
		if !ok {
			continue
		}
		clients = append(clients, catalogClient{
			ID:         c.ID,
			Kind:       c.Kind,
			Repo:       c.Repo,
			PinVersion: c.PinVersion,
			Toolchain:  c.Toolchain,
			LearnURL:   c.LearnURL,
		})
	}

	writeJSON(w, http.StatusOK, catalogResponse{Networks: networks, Clients: clients})
}

// ---------------------------------------------------------------------
// targets: list / add / delete
// ---------------------------------------------------------------------

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	targets := cfg.Targets
	if targets == nil {
		targets = []config.Target{}
	}
	writeJSON(w, http.StatusOK, targets)
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var t config.Target
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// Wire is only ever set by the setup wizard (POST .../setup), never
	// accepted directly from a client here.
	t.Wire = nil

	if t.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if t.Mode != "local" && t.Mode != "ssh" {
		writeError(w, http.StatusBadRequest, `mode must be "local" or "ssh"`)
		return
	}
	if t.Mode == "ssh" {
		if t.SSH == nil || t.SSH.Host == "" || t.SSH.User == "" || t.SSH.KeyPath == "" {
			writeError(w, http.StatusBadRequest, "ssh mode requires ssh.host, ssh.user, and ssh.keyPath")
			return
		}
		if t.SSH.HostKeyFile == "" {
			dir, err := config.Dir()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			t.SSH.HostKeyFile = filepath.Join(dir, "known_hosts")
		}
	}

	existing, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, ok := findTarget(existing, t.ID); ok {
		writeError(w, http.StatusConflict, fmt.Sprintf("target %q already exists", t.ID))
		return
	}

	// Dial now (SSH TOFU happens on this call) to validate the target is
	// reachable before persisting it.
	ex, err := s.newExecutor(t)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("could not reach target: %v", err))
		return
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		if _, ok := findTarget(*c, t.ID); ok {
			return fmt.Errorf("target %q already exists", t.ID)
		}
		c.Targets = append(c.Targets, t)
		return nil
	})
	if err != nil {
		ex.Close()
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	s.reg.get(t.ID).setExec(ex)

	added, _ := findTarget(cfg, t.ID)
	writeJSON(w, http.StatusCreated, added)
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	found := false
	_, err := s.updateConfig(func(c *config.Config) error {
		out := c.Targets[:0]
		for _, t := range c.Targets {
			if t.ID == id {
				found = true
				continue
			}
			out = append(out, t)
		}
		c.Targets = out
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}

	s.reg.remove(id)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// setup kickoff + SSE progress stream
// ---------------------------------------------------------------------

// validateWirePorts rejects any of WireConfig's port fields that fall
// outside 0..65535 — 0 means "use the default" (see catalog.WireConfig),
// so it's the only value below 1 that's allowed. The wizard UI validates
// the same range client-side, but the server can't trust that.
func validateWirePorts(wire catalog.WireConfig) error {
	ports := []struct {
		name string
		port int
	}{
		{"ExecHTTPPort", wire.ExecHTTPPort},
		{"BeaconHTTPPort", wire.BeaconHTTPPort},
		{"ExecP2PPort", wire.ExecP2PPort},
	}
	for _, p := range ports {
		if p.port < 0 || p.port > 65535 {
			return fmt.Errorf("%s: %d is out of range (must be 0 for default, or 1-65535)", p.name, p.port)
		}
	}
	return nil
}

func (s *Server) handleStartSetup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var wire catalog.WireConfig
	if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateWirePorts(wire); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if wire.DataDir == "" {
		wire.DataDir = fmt.Sprintf("/var/lib/valve-node/%d", wire.ChainID)
	}
	if wire.JWTPath == "" {
		wire.JWTPath = filepath.Join(wire.DataDir, "jwt.hex")
	}

	steps, err := setup.Plan(wire)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	entry := s.reg.get(id)
	entry.mu.Lock()
	if entry.setup != nil && entry.setup.running {
		entry.mu.Unlock()
		writeError(w, http.StatusConflict, "setup is already running for this target")
		return
	}
	setupCtx, setupCancel := context.WithCancel(context.Background())
	run := newSetupRun(setupCancel)
	entry.setup = run
	entry.mu.Unlock()

	// The wizard "has run" as soon as setup is kicked off, even if it fails
	// partway — the engine is idempotent (each step's Verify is also its
	// "already done" probe), so re-kicking setup against the same target
	// resumes rather than restarts. Persisting immediately also means the
	// UI can show what a target is (or was) being configured with even
	// while the run is still in flight.
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID == id {
				wireCopy := wire
				c.Targets[i].Wire = &wireCopy
			}
		}
		return nil
	}); err != nil {
		// Undo the "running" mark: the run never actually started, so a
		// retry must not be told setup is already in progress.
		entry.mu.Lock()
		entry.setup = nil
		entry.mu.Unlock()
		setupCancel()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	events := make(chan setup.Event, 32)
	go func() {
		for ev := range events {
			run.append(ev)
		}
	}()
	go func() {
		defer close(events)
		runErr := setup.RunAll(setupCtx, ex, steps, &setup.State{Wire: wire, Events: events})
		run.finish(runErr)
		// Signal that this goroutine is done touching ex — registry.remove
		// waits on this before Close()ing the executor out from under a
		// still-running setup step.
		close(run.done)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (s *Server) handleSetupStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	entry := s.reg.get(id)
	entry.mu.Lock()
	run := entry.setup
	entry.mu.Unlock()
	if run == nil {
		writeError(w, http.StatusNotFound, "no setup run has been started for this target")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	sseHeaders(w)

	snapshot, ch, unsub := run.subscribe()
	defer unsub()

	for _, ev := range snapshot {
		writeSSEEvent(w, ev)
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, ev)
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------
// per-target monitor stream
// ---------------------------------------------------------------------

func (s *Server) handleTargetMonitorStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return
	}

	mon, err := s.getMonitor(target, cfg.RefRPCBase)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	sseHeaders(w)

	ch, unsub := mon.Subscribe()
	defer unsub()

	writeSSEEvent(w, mon.Latest())
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, snap)
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------
// logs: recent + SSE tail
// ---------------------------------------------------------------------

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return
	}

	watch, err := s.getWatcher(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	n := defaultRecentLogs
	if raw := r.URL.Query().Get("n"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			n = parsed
		}
	}

	hits := watch.Recent(n)
	if hits == nil {
		hits = []logwatch.Hit{}
	}
	writeJSON(w, http.StatusOK, hits)
}

func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return
	}

	watch, err := s.getWatcher(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	sseHeaders(w)

	ch, unsub := watch.Subscribe()
	defer unsub()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case hit, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, hit)
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------
// explain
// ---------------------------------------------------------------------

// maxDefaultExplainHits caps how many recent error/critical log hits are
// sent to the AI provider when the caller doesn't supply explicit lines.
const maxDefaultExplainHits = 40

type explainRequest struct {
	Lines []string `json:"lines,omitempty"`
}

type explainResponse struct {
	Text string `json:"text"`
	// SentExcerpt is exactly the lines that were sent to the provider, so
	// the UI can show the operator what went out — whether that's the
	// caller-supplied lines or the auto-selected recent error hits.
	SentExcerpt []string `json:"sentExcerpt"`
}

func (s *Server) handleExplain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if cfg.AIProvider == "" {
		writeError(w, http.StatusConflict, "no AI provider is configured; set one in Settings first")
		return
	}

	var req explainRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	lines := req.Lines
	if len(lines) == 0 && target.Wire != nil {
		if watch, err := s.getWatcher(target); err == nil {
			for _, hit := range watch.Recent(0) {
				if hit.Severity == "error" || hit.Severity == "critical" {
					lines = append(lines, hit.Line)
				}
			}
			if len(lines) > maxDefaultExplainHits {
				lines = lines[len(lines)-maxDefaultExplainHits:]
			}
		}
	}

	provider, err := s.newAIProvider(cfg.AIProvider, cfg.AIKey, "")
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	var chainName, execID, beaconID string
	var syncing bool
	if target.Wire != nil {
		if net, ok := catalog.NetworkByChainID(target.Wire.ChainID); ok {
			chainName = net.Name
		}
		execID = target.Wire.ExecID
		beaconID = target.Wire.BeaconID
		if mon, err := s.getMonitor(target, cfg.RefRPCBase); err == nil {
			syncing = mon.Latest().ExecSyncing
		}
	}

	text, err := provider.Explain(r.Context(), ai.ExplainRequest{
		ChainName:    chainName,
		ExecClient:   execID,
		BeaconClient: beaconID,
		Syncing:      syncing,
		Lines:        lines,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, explainResponse{Text: text, SentExcerpt: lines})
}

// ---------------------------------------------------------------------
// service control: start/stop/restart, clear, disk usage, endpoints,
// firewall checklist — all day-2 operator actions from internal/ops,
// gated on the target existing and having completed setup (Wire != nil).
// ---------------------------------------------------------------------

// targetWithWire loads cfg, resolves id to a Target, and checks it has
// completed setup — the shared 404 (unknown target, checked first per the
// v0.1 review's ordering nit) / 409 (Wire == nil) preamble every route in
// this section needs before it can touch ops.
func (s *Server) targetWithWire(w http.ResponseWriter, r *http.Request, id string) (config.Target, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return config.Target{}, false
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return config.Target{}, false
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return config.Target{}, false
	}
	return target, true
}

// serviceActionResponse deliberately carries no json tag (like the ops
// structs it sits alongside) so it encodes as PascalCase {"Active":...},
// matching the spec's `{Active bool}`.
type serviceActionResponse struct {
	Active bool
}

func (s *Server) handleServiceAction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc := r.PathValue("svc")
	action := r.PathValue("action")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	active, err := ops.ServiceAction(r.Context(), ex, svc, action)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, serviceActionResponse{Active: active})
}

// clearRequest mirrors serviceActionResponse's untagged-field convention.
type clearRequest struct {
	Confirm string
}

func (s *Server) handleServiceClear(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc := r.PathValue("svc")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	var req clearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Confirm != svc {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("confirm must equal service name %q", svc))
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if err := ops.ClearService(r.Context(), ex, *target.Wire, svc); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	du, err := ops.DiskUsage(r.Context(), ex, *target.Wire)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, du)
}

func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	sshMode := target.Mode == "ssh"
	sshHostHint := ""
	if sshMode && target.SSH != nil {
		sshHostHint = fmt.Sprintf("%s@%s", target.SSH.User, target.SSH.Host)
	}

	ep, err := ops.Endpoints(r.Context(), ex, *target.Wire, sshMode, sshHostHint)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ep)
}

func (s *Server) handleFirewall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	items, err := ops.FirewallChecklist(r.Context(), ex, *target.Wire)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if items == nil {
		items = []ops.CheckItem{}
	}

	writeJSON(w, http.StatusOK, items)
}

// handleDiagnostics runs the network-diagnostics ladder manually (trigger
// "manual") and stores the result as the target's latest report. Auto runs
// (journal/monitor triggered — see diag.go) go through the same gate, so a
// manual click during an in-flight run gets a 409 rather than a duplicate
// probe storm.
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	entry := s.reg.get(id)
	if !entry.tryBeginDiag(time.Now(), false) {
		writeError(w, http.StatusConflict, "a diagnostics run is already in progress for this target")
		return
	}

	report, err := s.runDiagnostics(r.Context(), target, "manual")
	if err != nil {
		entry.endDiag(nil)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	entry.endDiag(report)

	writeJSON(w, http.StatusOK, report)
}

// handleDiagnosticsLatest returns the target's most recent diagnostics
// report — manual or auto-triggered — or JSON null when none has run yet.
func (s *Server) handleDiagnosticsLatest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, ok := s.targetWithWire(w, r, id); !ok {
		return
	}

	writeJSON(w, http.StatusOK, s.reg.get(id).latestDiag())
}

// ---------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------

type settingsResponse struct {
	AIProvider string `json:"aiProvider"`
	AIKeySet   bool   `json:"aiKeySet"`
	RefRPCBase string `json:"refRpcBase"`
}

func settingsResponseFrom(c config.Config) settingsResponse {
	return settingsResponse{
		AIProvider: c.AIProvider,
		AIKeySet:   c.AIKey != "",
		RefRPCBase: c.RefRPCBase,
	}
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settingsResponseFrom(cfg))
}

// settingsRequest uses pointer fields so PUT can distinguish "omitted,
// leave unchanged" from "explicitly set to empty" — most importantly for
// aiKey, which GET never echoes back, so a client re-PUTting the response
// of a prior GET must not blow away an already-stored key.
type settingsRequest struct {
	AIProvider *string `json:"aiProvider"`
	AIKey      *string `json:"aiKey"`
	RefRPCBase *string `json:"refRpcBase"`
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		if req.AIProvider != nil {
			c.AIProvider = *req.AIProvider
		}
		if req.AIKey != nil {
			c.AIKey = *req.AIKey
		}
		if req.RefRPCBase != nil {
			c.RefRPCBase = *req.RefRPCBase
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, settingsResponseFrom(cfg))
}
