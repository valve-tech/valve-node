package catalog

// clients is the source of truth for known execution/beacon clients.
// Repos and build commands are ported verbatim from
// packages/web/src/learn/data/clients.ts. Client ids and Kind values
// follow the exact naming fixed by task-3-brief.md: the upstream sigp
// lighthouse client is id "sigp-lighthouse" in clients.ts but must be id
// "lighthouse" here, and Kind is "exec"|"beacon" (clients.ts: "execution"|
// "consensus").
//
// None of the learn-site data publishes per-platform prebuilt-binary
// download URLs — only source-build (cargo) or docker-pull recipes, which
// are ported into BuildCmd. So every client's ReleaseURL is noReleaseURL:
// rather than invent binary URLs, callers always fall back to BuildCmd.
//
// PinVersion is not present anywhere in the learn data. Rather than invent
// a specific version number, each PinVersion below is read off the
// version identifier already literally present in that client's BuildCmd
// (the docker tag it pulls, or "main" for the two clients whose git-clone
// command names no tag/branch at all).
var clients = map[string]Client{
	"reth": {
		ID:         "reth",
		Kind:       "exec",
		Repo:       "https://github.com/valve-tech/reth",
		ReleaseURL: noReleaseURL,
		PinVersion: "main",
		BuildCmd:   "git clone https://github.com/valve-tech/reth && cd reth && cargo build --release --bin reth",
		LearnURL:   learnBaseURL,
	},
	"go-pulse": {
		ID:         "go-pulse",
		Kind:       "exec",
		Repo:       "https://gitlab.com/pulsechaincom/go-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		BuildCmd:   "docker pull registry.gitlab.com/pulsechaincom/go-pulse:latest",
		LearnURL:   learnBaseURL,
	},
	"erigon-pulse": {
		ID:         "erigon-pulse",
		Kind:       "exec",
		Repo:       "https://gitlab.com/pulsechaincom/erigon-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		BuildCmd:   "docker pull registry.gitlab.com/pulsechaincom/erigon-pulse:latest",
		LearnURL:   learnBaseURL,
	},
	"geth": {
		ID:         "geth",
		Kind:       "exec",
		Repo:       "https://github.com/ethereum/go-ethereum",
		ReleaseURL: noReleaseURL,
		PinVersion: "stable",
		BuildCmd:   "docker pull ethereum/client-go:stable",
		LearnURL:   learnBaseURL,
	},
	"lighthouse-pulse": {
		ID:         "lighthouse-pulse",
		Kind:       "beacon",
		Repo:       "https://github.com/valve-tech/lighthouse-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "main",
		BuildCmd:   "git clone https://github.com/valve-tech/lighthouse-pulse && cd lighthouse-pulse && RUSTUP_TOOLCHAIN=1.81.0 cargo build --release --bin lighthouse",
		LearnURL:   learnBaseURL,
	},
	"prysm-pulse": {
		ID:         "prysm-pulse",
		Kind:       "beacon",
		Repo:       "https://gitlab.com/pulsechaincom/prysm-pulse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		BuildCmd:   "docker pull registry.gitlab.com/pulsechaincom/prysm-pulse/beacon-chain:latest",
		LearnURL:   learnBaseURL,
	},
	"lighthouse": {
		ID:         "lighthouse",
		Kind:       "beacon",
		Repo:       "https://github.com/sigp/lighthouse",
		ReleaseURL: noReleaseURL,
		PinVersion: "latest",
		BuildCmd:   "docker pull sigp/lighthouse:latest",
		LearnURL:   learnBaseURL,
	},
}

// noReleaseURL is the shared ReleaseURL for every client in the catalog.
func noReleaseURL(goos, goarch, version string) string {
	return ""
}
