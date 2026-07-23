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

## Learn more

For a deeper guide to running your own RPC node, see
[learn.valve.city/rpc](https://learn.valve.city/rpc).

## License

[MIT](LICENSE)
