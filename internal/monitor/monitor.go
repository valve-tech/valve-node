// Package monitor polls a valve-node target's execution client, beacon
// client, disk, and systemd service state on a fixed interval, keeping the
// latest reading and fanning it out to subscribers (the SSE stream). Every
// target probe goes through an executor.Executor as a shell command (`curl
// -s -m 5 ...`, `df`, `systemctl is-active`) so it works identically whether
// the target is local or reached over SSH. The one exception is RefRPC, the
// public reference-head fetch, which talks net/http directly from the app
// host, not through the Executor.
//
// A failed probe (transport error, non-zero exit where that matters,
// unparseable JSON) never fails the whole poll: the affected Snapshot
// field(s) are left at their zero value and polling continues, since the
// target box may be mid-setup or between restarts.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// defaultInterval is used when Config.Interval is zero.
const defaultInterval = 5 * time.Second

// Target unit names, matching internal/catalog/units.go and
// internal/setup/steps.go's naming contract.
const (
	execUnitName   = "valve-node-exec.service"
	beaconUnitName = "valve-node-beacon.service"
)

// execRPCAddr and beaconAPIAddr resolve a poll's exec/beacon HTTP base
// URLs from the WireConfig's port resolvers (zero value = default 8545 /
// 5052), rather than hardcoding the defaults — a target configured with
// custom ports must be probed on those exact ports.
func execRPCAddr(w catalog.WireConfig) string {
	return fmt.Sprintf("http://%s:%d", w.RPCBind(), w.ExecHTTP())
}

func beaconAPIAddr(w catalog.WireConfig) string {
	return fmt.Sprintf("http://%s:%d", w.RPCBind(), w.BeaconHTTP())
}

// Snapshot is one point-in-time reading of a valve-node target's health.
type Snapshot struct {
	At             time.Time `json:"at"`
	ExecSyncing    bool      `json:"execSyncing"`
	ExecHead       uint64    `json:"execHead"`
	RefHead        uint64    `json:"refHead"` // public reference head; 0 = unavailable
	BeaconSlot     uint64    `json:"beaconSlot"`
	BeaconDistance uint64    `json:"beaconDistance"`
	ExecPeers      int       `json:"execPeers"`
	BeaconPeers    int       `json:"beaconPeers"`
	DiskUsedPct    float64   `json:"diskUsedPct"`
	ExecActive     bool      `json:"execActive"`
	BeaconActive   bool      `json:"beaconActive"`
}

// Config configures a Monitor.
type Config struct {
	Exec executor.Executor
	Wire catalog.WireConfig
	// RefRPC is the public reference endpoint for the chain, e.g.
	// https://rpc.valve.city/v1/<key>/evm/369 — head-lag baseline. Empty
	// disables the reference-head fetch (RefHead stays 0).
	RefRPC   string
	Interval time.Duration // 0 => 5s
}

// Monitor polls a target on Config.Interval and fans out each Snapshot to
// subscribers.
type Monitor struct {
	cfg Config

	httpClient *http.Client

	mu     sync.RWMutex
	latest Snapshot

	subsMu sync.Mutex
	subs   map[chan Snapshot]struct{}
}

// New constructs a Monitor from cfg. It does not start polling — call
// Start.
func New(cfg Config) *Monitor {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	return &Monitor{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		subs:       map[chan Snapshot]struct{}{},
	}
}

// Start begins polling in a background goroutine until ctx is canceled. An
// initial poll runs immediately (not after the first Interval tick).
func (m *Monitor) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *Monitor) run(ctx context.Context) {
	m.pollAndPublish(ctx)

	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollAndPublish(ctx)
		}
	}
}

func (m *Monitor) pollAndPublish(ctx context.Context) {
	snap := m.poll(ctx)

	m.mu.Lock()
	m.latest = snap
	m.mu.Unlock()

	m.publish(snap)
}

// Latest returns the most recently polled Snapshot (zero value if no poll
// has completed yet).
func (m *Monitor) Latest() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latest
}

// Subscribe registers a new subscriber and returns a channel that receives
// every subsequently published Snapshot, plus an unsubscribe func. The
// channel is best-effort: a slow consumer that doesn't drain it may miss
// ticks, but never blocks the poll loop. Callers must call the returned
// func when done to avoid leaking the subscription.
func (m *Monitor) Subscribe() (<-chan Snapshot, func()) {
	ch := make(chan Snapshot, 8)

	m.subsMu.Lock()
	m.subs[ch] = struct{}{}
	m.subsMu.Unlock()

	unsub := func() {
		m.subsMu.Lock()
		delete(m.subs, ch)
		m.subsMu.Unlock()
	}
	return ch, unsub
}

func (m *Monitor) publish(snap Snapshot) {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()
	for ch := range m.subs {
		select {
		case ch <- snap:
		default:
			// Slow consumer — drop this tick rather than block the poll
			// loop or other subscribers.
		}
	}
}

// poll gathers one Snapshot. Every probe is independently best-effort: a
// failure leaves its Snapshot field(s) zero and does not abort the rest of
// the poll.
func (m *Monitor) poll(ctx context.Context) Snapshot {
	snap := Snapshot{At: time.Now()}

	execAddr := execRPCAddr(m.cfg.Wire)
	beaconAddr := beaconAPIAddr(m.cfg.Wire)

	if res, err := m.cfg.Exec.Run(ctx, jsonRPCCmd(execAddr, "eth_syncing"), nil); err == nil && res.ExitCode == 0 {
		snap.ExecSyncing = parseEthSyncing(res.Stdout)
	}
	if res, err := m.cfg.Exec.Run(ctx, jsonRPCCmd(execAddr, "eth_blockNumber"), nil); err == nil && res.ExitCode == 0 {
		if head, ok := parseHexResult(res.Stdout); ok {
			snap.ExecHead = head
		}
	}
	if res, err := m.cfg.Exec.Run(ctx, jsonRPCCmd(execAddr, "net_peerCount"), nil); err == nil && res.ExitCode == 0 {
		if peers, ok := parseHexResult(res.Stdout); ok {
			snap.ExecPeers = int(peers)
		}
	}
	if res, err := m.cfg.Exec.Run(ctx, beaconSyncingCmd(beaconAddr), nil); err == nil && res.ExitCode == 0 {
		if slot, distance, ok := parseBeaconSyncing(res.Stdout); ok {
			snap.BeaconSlot = slot
			snap.BeaconDistance = distance
		}
	}
	if res, err := m.cfg.Exec.Run(ctx, beaconPeerCountCmd(beaconAddr), nil); err == nil && res.ExitCode == 0 {
		if peers, ok := parseBeaconPeerCount(res.Stdout); ok {
			snap.BeaconPeers = peers
		}
	}
	if res, err := m.cfg.Exec.Run(ctx, diskCmd(m.cfg.Wire.DataDir), nil); err == nil && res.ExitCode == 0 {
		if pct, ok := parseDiskPct(res.Stdout); ok {
			snap.DiskUsedPct = pct
		}
	}
	// systemctl is-active exits non-zero when a unit isn't active — that's
	// a valid (inactive) reading, not a probe failure, so only the
	// transport error is checked here.
	if res, err := m.cfg.Exec.Run(ctx, activeCmd(), nil); err == nil {
		snap.ExecActive, snap.BeaconActive = parseActive(res.Stdout)
	}

	snap.RefHead = m.fetchRefHead(ctx)

	return snap
}

// fetchRefHead fetches eth_blockNumber from the public reference RPC
// directly over net/http (not the Executor — this runs from the app host,
// not the target). Any failure (RefRPC unset, transport error, non-200,
// unparseable body) yields 0, not an error.
func (m *Monitor) fetchRefHead(ctx context.Context) uint64 {
	if m.cfg.RefRPC == "" {
		return 0
	}
	body := `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.RefRPC, strings.NewReader(body))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := m.httpClient.Do(req)
	if err != nil {
		return 0
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return 0
	}
	head, ok := parseHexResult(string(b))
	if !ok {
		return 0
	}
	return head
}

// ---------------------------------------------------------------------
// commands
// ---------------------------------------------------------------------

func jsonRPCCmd(addr, method string) string {
	return fmt.Sprintf(
		`curl -s -m 5 -X POST -H 'Content-Type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"%s","params":[]}' %s`,
		method, addr,
	)
}

func beaconSyncingCmd(addr string) string {
	return fmt.Sprintf("curl -s -m 5 %s/eth/v1/node/syncing", addr)
}

func beaconPeerCountCmd(addr string) string {
	return fmt.Sprintf("curl -s -m 5 %s/eth/v1/node/peer_count", addr)
}

func diskCmd(dir string) string {
	return fmt.Sprintf("df --output=pcent %s | tail -1", shQuote(dir))
}

func activeCmd() string {
	return fmt.Sprintf("systemctl is-active %s %s", execUnitName, beaconUnitName)
}

// shQuote single-quotes s for safe interpolation into a `sh -c` command
// string, escaping any embedded single quotes.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ---------------------------------------------------------------------
// parsing
// ---------------------------------------------------------------------

type jsonRPCResponse struct {
	Result json.RawMessage `json:"result"`
}

// parseHexResult parses a JSON-RPC response whose "result" is a 0x-hex
// string (eth_blockNumber, net_peerCount, and RefRPC's eth_blockNumber).
func parseHexResult(body string) (uint64, bool) {
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return 0, false
	}
	var hexStr string
	if err := json.Unmarshal(resp.Result, &hexStr); err != nil {
		return 0, false
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	if hexStr == "" {
		return 0, true
	}
	n, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseEthSyncing parses eth_syncing's response, whose "result" is either
// the literal false (not syncing) or an object (syncing).
func parseEthSyncing(body string) bool {
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return false
	}
	var b bool
	if err := json.Unmarshal(resp.Result, &b); err == nil {
		return b
	}
	// Any shape other than a bool (i.e. the sync-status object) means the
	// node is syncing.
	var obj map[string]any
	if err := json.Unmarshal(resp.Result, &obj); err == nil {
		return true
	}
	return false
}

type beaconSyncingResponse struct {
	Data struct {
		HeadSlot     string `json:"head_slot"`
		SyncDistance string `json:"sync_distance"`
	} `json:"data"`
}

func parseBeaconSyncing(body string) (slot, distance uint64, ok bool) {
	var resp beaconSyncingResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return 0, 0, false
	}
	slot, err1 := strconv.ParseUint(resp.Data.HeadSlot, 10, 64)
	distance, err2 := strconv.ParseUint(resp.Data.SyncDistance, 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return slot, distance, true
}

type beaconPeerCountResponse struct {
	Data struct {
		Connected string `json:"connected"`
	} `json:"data"`
}

func parseBeaconPeerCount(body string) (int, bool) {
	var resp beaconPeerCountResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return 0, false
	}
	n, err := strconv.ParseUint(resp.Data.Connected, 10, 64)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// parseDiskPct parses `df --output=pcent`'s output, which is a header line
// followed by a value line like " 42%" (percent sign, possibly padded).
func parseDiskPct(output string) (float64, bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimSuffix(line, "%")
		if f, err := strconv.ParseFloat(line, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// parseActive parses `systemctl is-active <exec-unit> <beacon-unit>`'s
// output: one status word per line, in the same order as the units were
// passed.
func parseActive(output string) (execActive, beaconActive bool) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	get := func(i int) bool { return i < len(lines) && strings.TrimSpace(lines[i]) == "active" }
	return get(0), get(1)
}
