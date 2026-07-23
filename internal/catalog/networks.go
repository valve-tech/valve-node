package catalog

// learnBaseURL is the LearnURL every Network and Client carries, per the
// plan decision that every catalog entry links back to the learn.valve.city
// runbook it is documented in.
const learnBaseURL = "https://learn.valve.city/rpc"

// networks is the source of truth for supported chains. Chain ids,
// checkpoint URLs, and the exec/beacon client ids valid on each are ported
// verbatim from packages/web/src/learn/data/networks.ts.
var networks = []Network{
	{
		ChainID:       1,
		Name:          "Ethereum",
		CheckpointURL: "https://beaconstate.ethstaker.cc",
		ExecClients:   []string{"reth", "geth"},
		BeaconClients: []string{"lighthouse"},
		LearnURL:      learnBaseURL,
	},
	{
		ChainID:       369,
		Name:          "PulseChain",
		CheckpointURL: "https://checkpoint.pulsechain.com",
		ExecClients:   []string{"reth", "go-pulse", "erigon-pulse"},
		BeaconClients: []string{"lighthouse-pulse", "prysm-pulse"},
		LearnURL:      learnBaseURL,
	},
	{
		ChainID:       943,
		Name:          "PulseChain Testnet v4",
		CheckpointURL: "https://checkpoint.v4.testnet.pulsechain.com",
		ExecClients:   []string{"reth", "go-pulse", "erigon-pulse"},
		BeaconClients: []string{"lighthouse-pulse", "prysm-pulse"},
		LearnURL:      learnBaseURL,
	},
}

// rethChainName maps a chain id to the --chain value reth expects, ported
// verbatim from the snapshot.rethChain values in networks.ts. erigon-pulse
// (the third independent PulseChain execution codebase) is assumed to
// share the same --chain naming convention as reth/go-pulse's fork family;
// this is not independently confirmed by learn data — see task-3-report.md.
var rethChainName = map[int]string{
	1:   "mainnet",
	369: "pulsechain",
	943: "pulsechain-testnet-v4",
}

// lighthouseNetworkName maps a chain id to the --network value the
// lighthouse client family (lighthouse-pulse and upstream sigp lighthouse)
// expects, per task-3-brief.md Step 3 verbatim.
var lighthouseNetworkName = map[int]string{
	1:   "mainnet",
	369: "pulsechain",
	943: "pulsechain_testnet_v4",
}
