# valve-node

**valve-node** sets up and monitors an Ethereum, PulseChain, or PulseChain-v4
node — one binary, guided setup, sync monitoring, and AI-generated log
explanations, all behind a token-gated local web UI.

- **Guided setup** — walks you through installing and wiring an execution
  client + consensus client for the network of your choice.
- **Sync monitoring** — watches your node while it syncs and once it's live,
  surfacing peer count, block height, and sync status at a glance.
- **AI log explanations** — turns cryptic client log lines into plain-English
  explanations of what's happening and whether you need to act.

Supported networks:

- **Ethereum** mainnet
- **PulseChain**
- **PulseChain v4** (testnet)

## v0.2

v0.2 rounds out day-to-day node operation from the same UI: start, stop, and
restart each service independently, or clear a service's data directory and
kick off a fresh resync — gated behind a typed confirmation so it can't
happen by accident. A storage panel compares actual disk usage against
expected-size estimates per client and network, labeled with rough
sync-time expectations. An endpoints panel lists each service's local RPC/P2P
URLs with live reachability checks, plus an SSH tunnel hint for reaching a
remote target's ports from your own machine. A security section runs a
probe-backed firewall checklist against the target and only ever *suggests*
the commands to lock it down — it never runs anything on your behalf. RPC and
P2P ports are configurable per client instead of fixed at their defaults.

## v0.3 (unreleased)

v0.3 de-roots the node services: the execution and beacon clients now run
as a dedicated unprivileged system user (`valve-node`) under hardened
systemd units (`NoNewPrivileges`, `ProtectSystem=strict` with the data
directory carved out, private `/tmp` and devices). Setup itself still
requires root — it creates the user, writes units, and owns the data
directory to the service account. Existing installs migrate automatically:
re-run setup against the target and the units are rewritten, the data
directory re-owned, and the services restarted.

## Requirements

- The **target** being set up (the box that will run the execution + beacon
  clients) must be **Debian or Ubuntu Linux**.
- The **SSH user must be root** — setup writes systemd units under
  `/etc/systemd/system`, manages services via `systemctl`, and installs
  binaries to `/usr/local/bin`. In **local mode** (setting up the same
  machine valve-node itself is running on), run valve-node as root. Preflight
  checks this (`id -u`) and fails fast with a clear message if it isn't met.
- Node services (the execution and beacon clients) run as the dedicated
  unprivileged `valve-node` system user, which setup creates. (In v0.1–v0.2
  they ran as root; re-running setup migrates an existing install.)

## Quickstart

### Download a release

Grab the binary for your platform from the
[latest release](https://github.com/valve-tech/valve-node/releases/latest),
then run it:

```bash
./valve-node
```

This prints a local URL with a one-time session token and opens it in your
browser:

```
http://127.0.0.1:8799/?token=<token>
```

Pass `--bind` to change the listen address, or `--no-open` to skip opening a
browser automatically.

### Build from source

Requires Go 1.25+ and Node 22+.

```bash
git clone https://github.com/valve-tech/valve-node.git
cd valve-node
cd cmd/valve-node/web && npm ci && npm run build && cd ../../..
go build -o valve-node ./cmd/valve-node
./valve-node
```

## How it's built

valve-node is a single Go binary with the web UI (Vite + TypeScript)
compiled to static assets and embedded directly into the binary via
`go:embed`. There's no separate frontend server and no external dependency
to run — just the binary.

The local server binds to `127.0.0.1` by default and requires a session
token for every request (via `Authorization: Bearer`, a cookie set from the
initial `?token=` link, or the query parameter itself), so nothing on your
machine can drive it without that token.

valve-node itself always runs locally — the UI and API bind to your own
machine. What it sets up can be **local** (the same machine) or **remote
over SSH**: point a target at a `host:port` + root SSH credentials and
valve-node drives the whole install/wire/start/handshake flow on that box
instead, so you can run valve-node on a laptop while it provisions a
dedicated server. Both modes need root on the target for setup itself (see
Requirements above); the node services it installs run unprivileged.

## Contributing

The web UI (`cmd/valve-node/web/`) has no end-to-end (Playwright) test suite
by design for v1 — the API layer it talks to (`internal/server`) is fully
covered by Go tests, and the UI itself is a thin, framework-free render
layer over that API. Verify UI changes with:

```bash
cd cmd/valve-node/web && npm run build   # tsc --noEmit (strict) + vite build
go build ./...                            # confirms the rebuilt dist/ still embeds
```

then a manual smoke test: run `./valve-node --no-open` against a scratch
`$HOME` and curl the printed token URL.

## Learn more

For a deeper guide to running your own RPC node, see
[learn.valve.city/rpc](https://learn.valve.city/rpc).

## License

[MIT](LICENSE)
