// Package catalog is the pure-data knowledge base for valve-node: the
// networks it knows how to configure (Ethereum, PulseChain, PulseChain
// Testnet v4), the execution/beacon clients it knows how to run on each,
// and the systemd unit templates that pair an execution client with a
// beacon client for a given network. It performs no I/O and never touches
// an executor.Executor — writing the rendered units to disk is Task 4's
// job.
//
// Network and client data (chain ids, checkpoint URLs, client repos and
// build commands) are ported verbatim from the learn.valve.city runbook
// data in the monorepo (packages/web/src/learn/data/{networks,clients}.ts).
package catalog

import "fmt"

// Network describes one of the chains valve-node can configure an
// execution+beacon client pair for.
type Network struct {
	ChainID       int    // 1 | 369 | 943
	Name          string // "Ethereum" | "PulseChain" | "PulseChain Testnet v4"
	CheckpointURL string
	ExecClients   []string // client ids valid as the execution client on this chain
	BeaconClients []string // client ids valid as the beacon client on this chain
	LearnURL      string

	// ArchiveSizeTB is the archive-tier dataset size in terabytes, ported
	// verbatim from learn.valve.city's snapshot.sizeTB. The full(pruned)
	// tier is estimated as half this value — see ExpectedBytes.
	ArchiveSizeTB float64
	// SyncLabel and GenesisSyncLabel are the human sync-time estimates
	// shown on learn.valve.city — SyncLabel for a snapshot-assisted sync,
	// GenesisSyncLabel for a from-genesis sync.
	SyncLabel        string
	GenesisSyncLabel string
}

// ExpectedBytes returns the expected on-disk dataset size, in bytes, for a
// chain at either the archive or full(pruned) tier. This is the single
// shared implementation of the size heuristic — setup's preflight disk
// check imports it rather than keeping its own copy. The full tier is
// estimated as half the archive tier's size; there is no learn-data source
// for a full-tier figure (see chainArchiveSizeTB's original comment in
// setup/steps.go, now folded in here).
func ExpectedBytes(chainID int, archive bool) (uint64, error) {
	net, ok := NetworkByChainID(chainID)
	if !ok {
		return 0, fmt.Errorf("catalog: no size guidance for chain id %d", chainID)
	}
	sizeTB := net.ArchiveSizeTB
	if !archive {
		sizeTB /= 2
	}
	return uint64(sizeTB * 1e12), nil
}

// Client describes one execution or beacon client valve-node knows how to
// obtain and wire up.
type Client struct {
	ID   string // "reth" "go-pulse" "erigon-pulse" "geth" "lighthouse-pulse" "prysm-pulse" "lighthouse"
	Kind string // "exec" | "beacon"
	Repo string // canonical source URL

	// ReleaseURL returns a prebuilt-binary URL for goos/goarch, "" if the
	// project publishes none for that platform (=> source build).
	ReleaseURL func(goos, goarch, version string) string

	PinVersion string // known-good default version tag
	BuildCmd   string // source-build recipe: a full sh script, run in a fresh
	// working dir, that ends with the binary installed executable at
	// /usr/local/bin/<ID> (matching what setup's install-step Verify checks).
	Toolchain string // "go" | "rust" — the build toolchain BuildCmd needs
	LearnURL  string

	// DataSubdirs lists the path(s), relative to a WireConfig's DataDir,
	// that this client owns exclusively — the data a "clear & resync"
	// deletes for this client and no other (v0.2 spec §2). Some clients
	// (geth-family, reth, erigon) write these subdirs implicitly under a
	// --datadir that IS the shared DataDir; others (prysm/lighthouse
	// families) are given a --datadir that already points at their own
	// subdir. Either way DataSubdirs names the on-disk owned path(s), and
	// RenderUnits' datadir flags must agree with it (see the golden
	// agreement tests in catalog_test.go).
	DataSubdirs []string
}

// Networks returns the full catalog of supported chains.
func Networks() []Network {
	out := make([]Network, len(networks))
	copy(out, networks)
	return out
}

// NetworkByChainID looks up a network by its chain id.
func NetworkByChainID(id int) (Network, bool) {
	for _, n := range networks {
		if n.ChainID == id {
			return n, true
		}
	}
	return Network{}, false
}

// ClientByID looks up a client by its id.
func ClientByID(id string) (Client, bool) {
	c, ok := clients[id]
	return c, ok
}
