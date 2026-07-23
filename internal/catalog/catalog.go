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

// Network describes one of the chains valve-node can configure an
// execution+beacon client pair for.
type Network struct {
	ChainID       int    // 1 | 369 | 943
	Name          string // "Ethereum" | "PulseChain" | "PulseChain Testnet v4"
	CheckpointURL string
	ExecClients   []string // client ids valid as the execution client on this chain
	BeaconClients []string // client ids valid as the beacon client on this chain
	LearnURL      string
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
	BuildCmd   string // source-build fallback, run in a clone dir
	LearnURL   string
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
