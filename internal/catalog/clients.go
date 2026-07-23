package catalog

// clients is the source of truth for known execution/beacon clients.
// Client ids and Kind values follow the exact naming fixed by
// task-3-brief.md: the upstream sigp lighthouse client is id
// "sigp-lighthouse" in clients.ts but must be id "lighthouse" here, and
// Kind is "exec"|"beacon" (clients.ts: "execution"|"consensus").
//
// BuildCmd is a source-build recipe (task-4b-brief.md): a full sh script,
// run via Executor.Run in a fresh working dir, that clones the client's
// repo at depth 1 and ends with a runnable binary installed at
// /usr/local/bin/<client-id> — the exact path setup's install-step Verify
// probes (`test -x /usr/local/bin/<id> && /usr/local/bin/<id> --version`).
// This replaces the earlier docker-pull/no-op recipes ported from the
// learn docs, which never produced a binary setup could actually run.
//
// Toolchain names the build toolchain BuildCmd needs ("go" or "rust"); the
// setup package's toolchain step reads it to decide what to bootstrap on
// the target before install runs.
//
// None of the learn-site data publishes per-platform prebuilt-binary
// download URLs, so every client's ReleaseURL is noReleaseURL: rather than
// invent binary URLs, callers always fall back to BuildCmd.
//
// PinVersion is not present anywhere in the learn data. Rather than invent
// a specific version number, each PinVersion below is read off the
// version identifier already literally present in the client's old
// docker-pull BuildCmd (the docker tag it pulled, or "main" for the two
// clients whose git-clone command named no tag/branch at all) — BuildCmd
// itself now clones at depth 1 off the repo's default branch, so
// PinVersion is descriptive metadata only, not consulted by BuildCmd.
var clients = map[string]Client{
	"reth": {
		ID:         "reth",
		Kind:       "exec",
		Repo:       "https://github.com/valve-tech/reth",
		ReleaseURL: noReleaseURL,
		PinVersion: "main",
		Toolchain:  "rust",
		BuildCmd: `rm -rf /tmp/build-reth && ` +
			`git clone --depth 1 https://github.com/valve-tech/reth.git /tmp/build-reth && ` +
			`cd /tmp/build-reth && ` +
			`(. "$HOME/.cargo/env" 2>/dev/null || true) && ` +
			`cargo build --release --bin reth && ` +
			`install -m 0755 target/release/reth /usr/local/bin/reth && ` +
			`/usr/local/bin/reth --version`,
		LearnURL: learnBaseURL,
	},
	"go-pulse": {
		ID:         "go-pulse",
		Kind:       "exec",
		Repo:       "https://gitlab.com/pulsechaincom/go-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		Toolchain:  "go",
		BuildCmd: `rm -rf /tmp/build-go-pulse && ` +
			`git clone --depth 1 https://gitlab.com/pulsechaincom/go-pulse.git /tmp/build-go-pulse && ` +
			`cd /tmp/build-go-pulse && ` +
			`go build -o /usr/local/bin/go-pulse ./cmd/geth && ` +
			`/usr/local/bin/go-pulse version`,
		LearnURL: learnBaseURL,
	},
	"erigon-pulse": {
		ID:         "erigon-pulse",
		Kind:       "exec",
		Repo:       "https://gitlab.com/pulsechaincom/erigon-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		Toolchain:  "go",
		BuildCmd: `rm -rf /tmp/build-erigon-pulse && ` +
			`git clone --depth 1 https://gitlab.com/pulsechaincom/erigon-pulse.git /tmp/build-erigon-pulse && ` +
			`cd /tmp/build-erigon-pulse && ` +
			`go build -o /usr/local/bin/erigon-pulse ./cmd/erigon && ` +
			`/usr/local/bin/erigon-pulse --version`,
		LearnURL: learnBaseURL,
	},
	"geth": {
		ID:         "geth",
		Kind:       "exec",
		Repo:       "https://github.com/ethereum/go-ethereum",
		ReleaseURL: noReleaseURL,
		PinVersion: "stable",
		Toolchain:  "go",
		BuildCmd: `rm -rf /tmp/build-geth && ` +
			`git clone --depth 1 https://github.com/ethereum/go-ethereum.git /tmp/build-geth && ` +
			`cd /tmp/build-geth && ` +
			`go build -o /usr/local/bin/geth ./cmd/geth && ` +
			`/usr/local/bin/geth version`,
		LearnURL: learnBaseURL,
	},
	"lighthouse-pulse": {
		ID:         "lighthouse-pulse",
		Kind:       "beacon",
		Repo:       "https://github.com/valve-tech/lighthouse-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "main",
		Toolchain:  "rust",
		BuildCmd: `rm -rf /tmp/build-lighthouse-pulse && ` +
			`git clone --depth 1 https://gitlab.com/pulsechaincom/lighthouse-pulse.git /tmp/build-lighthouse-pulse && ` +
			`cd /tmp/build-lighthouse-pulse && ` +
			`(. "$HOME/.cargo/env" 2>/dev/null || true) && ` +
			`cargo build --release --bin lighthouse && ` +
			`install -m 0755 target/release/lighthouse /usr/local/bin/lighthouse-pulse && ` +
			`/usr/local/bin/lighthouse-pulse --version`,
		LearnURL: learnBaseURL,
	},
	"prysm-pulse": {
		ID:         "prysm-pulse",
		Kind:       "beacon",
		Repo:       "https://gitlab.com/pulsechaincom/prysm-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		Toolchain:  "go",
		BuildCmd: `rm -rf /tmp/build-prysm-pulse && ` +
			`git clone --depth 1 https://gitlab.com/pulsechaincom/prysm-pulse.git /tmp/build-prysm-pulse && ` +
			`cd /tmp/build-prysm-pulse && ` +
			`go build -o /usr/local/bin/prysm-pulse ./cmd/beacon-chain && ` +
			`/usr/local/bin/prysm-pulse --version`,
		LearnURL: learnBaseURL,
	},
	"lighthouse": {
		ID:         "lighthouse",
		Kind:       "beacon",
		Repo:       "https://github.com/sigp/lighthouse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		Toolchain:  "rust",
		BuildCmd: `rm -rf /tmp/build-lighthouse && ` +
			`git clone --depth 1 https://github.com/sigp/lighthouse.git /tmp/build-lighthouse && ` +
			`cd /tmp/build-lighthouse && ` +
			`(. "$HOME/.cargo/env" 2>/dev/null || true) && ` +
			`cargo build --release --bin lighthouse && ` +
			`install -m 0755 target/release/lighthouse /usr/local/bin/lighthouse && ` +
			`/usr/local/bin/lighthouse --version`,
		LearnURL: learnBaseURL,
	},
}

// noReleaseURL is the shared ReleaseURL for every client in the catalog.
func noReleaseURL(goos, goarch, version string) string {
	return ""
}
