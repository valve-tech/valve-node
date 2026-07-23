# Network Diagnostics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A one-click "Network diagnostics" report that tells the operator what is wrong with their node's network stack (services, RPC, listeners, inbound/outbound reachability, peers, sync, journal error signatures) and how to fix it.

**Architecture:** A new `ops.NetworkDiagnostics(ctx, e, w, opts)` composes ~9 ordered read-only probes into the existing `ops.CheckItem` finding shape (ID/Title/Why/Status/Detail/Fix), reusing `bindState`/`beaconP2PPorts`/`p2pOpenItem`/`parseHexResult` and the monitor's curl command shapes. Journal lines are cross-referenced against `logwatch`'s signature table via a newly exported `logwatch.Classify`. In SSH mode an inbound-reachability probe dials the target's public P2P ports from the app host (injected dialer for tests). Served at `GET /api/targets/{id}/diagnostics` next to the firewall route; rendered by a new `#/diag/<id>` screen cloned from `security.ts`, linked from the dashboard.

**Tech Stack:** Go 1.25, existing fakeExecutor/scriptedExecutor test doubles, Vite+TS UI (build-only verification per README Contributing).

## Global Constraints

- Probes are strictly read-only on the target; every remedy is copy-paste text in `CheckItem.Fix` (same contract as `FirewallChecklist`, ops.go:445-449).
- All target commands go through `executor.Executor`; curl probes use the established `-s -m 5` shapes from `ops.Endpoints`/`internal/monitor`.
- Findings reuse `ops.CheckItem` verbatim (PascalCase JSON, Status ∈ pass|fail|warn|unknown) so the security-screen card renderer pattern carries over.
- No import cycles: ops → logwatch is new but safe (logwatch imports only executor).
- Verify: `go build ./... && go test ./...`; UI via `cd cmd/valve-node/web && npm run build` then `go build ./...` re-embed check.

---

### Task 1: `logwatch.Classify` — export line classification

**Files:** Modify `internal/logwatch/logwatch.go` (wrap unexported `classify`, line 194). Test: `internal/logwatch/logwatch_test.go`.

**Interfaces:** Produces `func Classify(unit, line string, now time.Time) (Hit, bool)` — identical semantics to the unexported `classify` (signature table first-match-wins; unmatched error-level lines yield an unclassified Hit; benign lines yield ok=false).

Steps: failing test (`TestClassify_Exported`: a `low peer count`-matching line returns Signature "low-peer-count" with non-empty Explain; a benign line returns ok=false) → red → implement as a one-line exported wrapper → green → commit `feat(logwatch): export Classify for cross-package signature matching`.

### Task 2: `ops.NetworkDiagnostics` — the probe suite

**Files:** Create `internal/ops/diagnose.go`, `internal/ops/diagnose_test.go`.

**Interfaces:**

```go
// DiagnoseOpts carries the app-host-side context probes need beyond the
// executor: SSH mode/host for the inbound dial probe, and an injectable
// dialer so tests never open real sockets.
type DiagnoseOpts struct {
	SSHMode bool
	SSHHost string // bare host/IP (no user@); "" skips the inbound probe
	Dial    func(network, addr string, timeout time.Duration) error // nil = net.DialTimeout
}
func NetworkDiagnostics(ctx context.Context, e executor.Executor, w catalog.WireConfig, opts DiagnoseOpts) ([]CheckItem, error)
```

Probe order and classification (each one CheckItem; IDs stable for the UI):

1. `diag-services` — `systemctl is-active valve-node-exec.service valve-node-beacon.service` (line-per-unit, exit code ignored, like `ownActiveUnitPorts`). Both active → pass; else fail, Fix `systemctl start <unit>` + `journalctl -u <unit> -n 50 --no-pager`. Later probes that depend on a running service report `unknown` (not fail) when it isn't active.
2. `diag-exec-rpc` — the Endpoints eth_chainId curl. Reachable + chain matches → pass; reachable + mismatch → fail ("wrong network"); unreachable → fail if exec active ("running but not answering RPC — likely still starting; check journal"), else unknown.
3. `diag-beacon-api` — `/eth/v1/node/version` curl, `200` → pass; else fail/unknown by beacon-active.
4. `exec-p2p-open` / `beacon-p2p-open` — reuse `p2pOpenItem` on fresh `ss -ltn`/`ss -lun` output (same IDs as the firewall checklist — same probes, one coherent report).
5. `diag-p2p-inbound` (only when `opts.SSHMode && opts.SSHHost != ""`) — dial `tcp host:ExecP2P()` and `tcp host:<beacon TCP>` with 3s timeout from the app host. Both connect → pass; any fail → warn (provider firewall / NAT / ufw), Fix = ufw allow lines + "check your hosting provider's firewall".
6. `diag-outbound` — from the target: `curl -s -m 8 -o /dev/null <Network.CheckpointURL>`; exit 0 → pass (DNS+TLS+outbound OK); exit 6 → fail "DNS resolution failed"; 7 → fail "connection refused/blocked"; 28 → fail "timed out (egress filtered?)"; other → warn with exit code.
7. `diag-exec-peers` — `net_peerCount`: 0 → fail; 1–4 → warn; ≥5 → pass; unreadable → unknown. Detail includes the count; Fix (fail/warn) points at p2p port/firewall items and notes peer discovery takes a few minutes after start.
8. `diag-beacon-peers` — `/eth/v1/node/peer_count` `data.connected` (string per beacon API): 0 → fail; 1–9 → warn; ≥10 → pass; unreadable → unknown.
9. `diag-sync` — `eth_syncing` (result false = synced) + `/eth/v1/node/syncing` (`is_syncing`, `sync_distance`): both synced → pass; either syncing → warn with head/distance detail and "normal during initial sync" copy; unreadable → unknown.
10. `diag-journal` — `journalctl -u valve-node-exec.service -u valve-node-beacon.service -n 200 --no-pager -o cat`, classify each line with `logwatch.Classify`, count named signatures. None → pass; any error/critical signature → fail; warn-only → warn. Detail = distinct signatures with counts + the worst signature's canned Explain (+ LearnURL); Fix directs to the Logs screen for AI explanations.

Parsers `parseEthSyncingResult`, `parseBeaconSyncing`, `parseBeaconPeerCount` are package-local (duplication precedent: `parseHexResult` comment, ops.go:407-412).

Tests (fakeExecutor, table style like ops_test.go): happy path all-pass; services-inactive cascades unknown; 0 exec peers → fail with count in Detail; chain-id mismatch → fail; outbound curl exit 6 → DNS fail; journal with a `low peer count` line → warn naming the signature; inbound dial probe called only in SSH mode and warn on dial error (fake dialer records addrs); command list is strictly read-only (assert no mutating verbs in callLog).

Commit: `feat(ops): network diagnostics probe suite`.

### Task 3: server route

**Files:** Modify `internal/server/api.go` (route table ~line 411, handler next to `handleFirewall` ~line 1151). Test: `internal/server/api_test.go`.

Route: `mux.HandleFunc("GET /api/targets/{id}/diagnostics", s.handleDiagnostics)`. Handler mirrors `handleFirewall`: `targetWithWire` → `getExecutor` → build `ops.DiagnoseOpts{SSHMode: target.Mode == "ssh", SSHHost: host-from-target.SSH}` (nil Dial = real dialer) → 502 on probe error → `writeJSON` the items (empty slice, never null). Tests: happy path via `newAPITestServerWithExecutor` + `scriptedExecutor` (local-mode target → no dialing), unknown target 404, wire-less target 409.

Commit: `feat(server): GET /api/targets/{id}/diagnostics route`.

### Task 4: UI screen

**Files:** Create `cmd/valve-node/web/src/diag.ts` (clone of `security.ts` shape). Modify: `api.ts` (add `getNetworkDiagnostics(id)` reusing the firewall `CheckItem` type), `main.ts` (hash whitelist + route case `diag`), `dashboard.ts` (add `Diagnostics →` card-link next to `Security →`). Rebuild `web/dist` + go re-embed.

Copy tone matches security.ts: "Live, read-only probes run against the target — nothing is changed automatically. Run them when peers are low, sync is stuck, or you suspect a network problem." Re-run button re-fetches.

Verify: `cd cmd/valve-node/web && npm run build` (tsc strict + vite) then `go build ./... && go test ./...`.

Commit: `feat(ui): network diagnostics screen`.

### Task 5: README

Add v0.3 bullet describing network diagnostics. Commit: `docs: network diagnostics in v0.3 overview`.

## Addendum (user feedback, same day)

Implemented after the initial five tasks, per the user's direction that diagnostics "should happen when a log triggers it or when a connection fails" and read as "check check check, failed here":

- **Ladder semantics** — `NetworkDiagnostics` stops at the first `fail`; the returned slice ends at the failing rung. Warns don't stop it. The dead "service inactive → unknown" branches in later probes were removed (unreachable once a services failure stops the run).
- **Auto-trigger** (`internal/server/diag.go`) — per-target goroutines subscribe to the existing logwatch and monitor streams (started when those are first created): error/critical journal hits and failed-connection snapshots (service inactive, zero peers) start a background run, gated by a 10-minute cooldown + singleflight. Results stored as the target's latest `DiagReport {at, trigger, items, failedId}`.
- **API** — `GET /diagnostics` (manual; stores + returns a `DiagReport`, 409 while a run is in flight) and `GET /diagnostics/latest` (stored report or JSON null).
- **UI** — the diag screen shows the latest report (auto or manual) with the trigger and timestamp, the failing rung badged "failed here" with its fix expanded, and a note that later checks were skipped.

## Self-Review

Coverage: services/RPC/listeners/inbound/outbound/peers/sync/journal — the full "what's wrong + how to solve" ladder; UI + API + docs tasked. Types consistent: `CheckItem` reused everywhere; `DiagnoseOpts.Dial` injected for tests; `logwatch.Classify` exported in Task 1 and consumed in Task 2. Judgment calls: same IDs reused for the two shared p2p probes; thresholds (5 exec / 10 beacon peers) are deliberately loose to avoid noise; beacon P2P port knowledge stays in ops (`beaconP2PPorts`), the catalog gap noted but not addressed here.
