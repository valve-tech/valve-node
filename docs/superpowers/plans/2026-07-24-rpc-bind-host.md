# Node RPC Bind-to-Host (Tailscale reach) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Let the operator bind the execution + beacon HTTP RPC to a chosen host address (default loopback) so a node is reachable over Tailscale / other overlay networks from a desktop, instead of only through the SSH tunnel.

**Architecture:** A new `WireConfig.RPCBindAddr` (default `127.0.0.1`, resolved via `RPCBind()`) is threaded into the exec `--http.addr`, beacon `--http-address`, and prysm `--grpc-gateway-host` flags. The engine API (`--authrpc.addr`, 8551) and prysm's native `--rpc-host` stay hardcoded loopback — never exposed. Because a client binding to a single non-loopback address stops listening on loopback, every on-box probe address (monitor, diagnostics, ops.Endpoints, setup handshake) switches from a hardcoded `127.0.0.1` to `RPCBind()` — backward-compatible since the default is `127.0.0.1`. The firewall/diagnostics checklist gains bind-tier grading so a deliberate Tailscale bind reads as pass-with-note rather than the blanket `fail` any non-loopback bind gets today.

**Tech Stack:** Go 1.25, scripted fake-executor table tests, Vite+TS UI.

## Global Constraints

- Default `RPCBindAddr` is `""` → `RPCBind()` returns `127.0.0.1`; existing configs and all current behavior are unchanged.
- Engine API bind (`--authrpc.addr 127.0.0.1 --authrpc.port 8551`) and prysm `--rpc-host=127.0.0.1` are NEVER made configurable.
- Security tiers for a bind address: loopback (`127.0.0.1`/`::1`) → pass; Tailscale CGNAT `100.64.0.0/10` → pass-with-note; other RFC1918 (`10/8`,`172.16/12`,`192.168/16`) → warn (LAN exposure); anything else incl. `0.0.0.0` → fail (public/unauthenticated exposure).
- exec JSON-RPC is unauthenticated — the UI must state that binding it to a tailnet trusts everyone on the tailnet.
- Verify: `go build ./... && go test ./...`; UI via `cd cmd/valve-node/web && npm run build` then `go build ./...`.

## Security decision baked in (Option A, flag if you want it changed)

Bind to a **specific** host IP (the node's Tailscale address), not `0.0.0.0`-plus-firewall. Least exposure, no reliance on a correctly-configured firewall, consistent with this codebase's security-first posture. Operational caveat surfaced in UI/docs: the chosen interface must be up when the service starts (Tailscale before the node), else the client fails to bind.

---

### Task 1: catalog — `RPCBindAddr` field + thread into HTTP bind flags

**Files:** Modify `internal/catalog/units.go` (WireConfig ~line 11, `execCommand`, `beaconCommand`). Test: `internal/catalog/catalog_test.go`.

**Interfaces:** Produces `WireConfig.RPCBindAddr string` + `func (w WireConfig) RPCBind() string` (returns `"127.0.0.1"` when empty). The exec `--http.addr`, beacon `--http-address`, prysm `--grpc-gateway-host` render `RPCBind()`; `--authrpc.addr`/`--rpc-host` stay `127.0.0.1`.

- [ ] Failing test: with `RPCBindAddr: "100.101.102.103"`, each exec unit's `--http.addr 100.101.102.103` and beacon `--http-address`/`--grpc-gateway-host=100.101.102.103` appear, while `--authrpc.addr 127.0.0.1` (and prysm `--rpc-host=127.0.0.1`) remain loopback. With the field empty, everything still renders `127.0.0.1`.
- [ ] Run → red.
- [ ] Implement: add field + `RPCBind()`; swap the six `--http.addr 127.0.0.1`/`--http-address 127.0.0.1`/`--grpc-gateway-host=127.0.0.1` literals for `%s` + `w.RPCBind()`. Leave authrpc/rpc-host literals.
- [ ] Run package tests → green (existing port/flag substring tests still pass since default is `127.0.0.1`).
- [ ] Commit `feat(catalog): configurable RPC bind address (default loopback), threaded into exec/beacon HTTP flags`.

### Task 2: on-box probe addresses follow the bind

**Files:** `internal/monitor/monitor.go` (execRPCAddr/beaconAPIAddr), `internal/ops/diagnose.go` (exec/beacon addrs), `internal/ops/ops.go` (Endpoints), `internal/setup/steps.go` (handshake probes). Tests in each package.

**Interfaces:** Each `http://127.0.0.1:<port>` becomes `http://<w.RPCBind()>:<port>`. Default unchanged.

- [ ] Failing tests where feasible (monitor/diagnose/endpoints assert the probed address uses a custom `RPCBind()`); red; implement; green.
- [ ] Commit `fix(monitor,ops,setup): on-box probes target the configured RPC bind address, not a hardcoded loopback`.

### Task 3: bind-tier grading in the firewall/diagnostics checklist

**Files:** `internal/ops/ops.go` (`rpcNotPublicItem`, and a new `bindTier` helper), `internal/ops/diagnose.go` if needed. Tests: `internal/ops/ops_test.go`.

**Interfaces:** New `func bindTier(addr string) (status, note string)` classifying loopback/tailscale/rfc1918/public. `rpcNotPublicItem` reports the configured bind's tier for exec+beacon HTTP (engine always loopback so always fine): loopback pass, tailscale pass+note, rfc1918 warn, public fail. Detail names the tier; Fix (warn/fail) explains overlay-vs-public and the unauthenticated-RPC caveat.

- [ ] Failing tests per tier; red; implement; green.
- [ ] Commit `feat(ops): grade RPC bind exposure (loopback/tailscale/LAN/public) in the checklist`.

### Task 4: validation + config plumbing

**Files:** `internal/server/api.go` (`validateWire*`), `internal/config` if the field needs persisting (it rides on `WireConfig`, already persisted). Test: `internal/server/api_test.go`.

**Interfaces:** Reject a non-empty `RPCBindAddr` that isn't a valid IP. Empty allowed (default).

- [ ] Failing test (bad IP → 400); red; implement with `net.ParseIP`; green.
- [ ] Commit `feat(server): validate RPCBindAddr`.

### Task 5: UI — wizard field, endpoints reachable URL, caveat copy

**Files:** `cmd/valve-node/web/src/wizard.ts` (bind field), `dashboard.ts` (endpoints card shows the routable URL + drops the tunnel hint when bound routable), `api.ts` (WireConfig type). Rebuild dist + re-embed.

- [ ] Add an optional "RPC bind address" field (placeholder `127.0.0.1`, help text: use your Tailscale IP to reach the node from another machine; warns that exec RPC is unauthenticated so only bind to a trusted overlay). Endpoints card: when `RPCBind()` is routable, show `http://<addr>:<port>` as the reachable URL and hide the SSH tunnel hint.
- [ ] `npm run build` + `go build`; commit `feat(ui): RPC bind address in the wizard; endpoints show the reachable URL`.

### Task 6: docs

**Files:** `README.md`. Add a bind-to-host / Tailscale paragraph to the v0.3 section. Commit `docs: node RPC bind-to-host for Tailscale reach`.

## Self-Review

Coverage: config+render (1), on-box probe correctness — the loopback-probe trap (2), security grading (3), validation (4), UX incl. the unauthenticated caveat (5), docs (6). Default-loopback everywhere keeps existing installs byte-identical. Engine + prysm rpc-host stay loopback by construction. Open follow-up (noted, not built): ordering the node service `After=tailscaled` so binding to a tailnet IP doesn't race the interface coming up — deferred to the Phase-2 platform work.
