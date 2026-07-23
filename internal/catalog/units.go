package catalog

import (
	"bytes"
	"fmt"
	"path"
	"text/template"
)

// WireConfig describes one execution+beacon pairing to render systemd
// units for.
type WireConfig struct {
	ChainID          int
	ExecID, BeaconID string
	DataDir          string // e.g. /var/lib/valve-node/369
	JWTPath          string // <DataDir>/jwt.hex
	Archive          bool
}

// engineEndpoint is the local engine-API (JSON-RPC over HTTP, JWT-authed)
// endpoint the beacon client talks to the execution client on. Fixed by
// the plan decision — always loopback, always 8551.
const engineEndpoint = "http://127.0.0.1:8551"

// execHTTPPort and beaconHTTPPort are the plan-decided ports for each
// client's own HTTP RPC API (distinct from the engine endpoint above).
const execHTTPPort = "8545"
const beaconHTTPPort = "5052"

const unitTemplate = `[Unit]
Description=valve-node {{.Description}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.ExecStart}}
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`

type unitVars struct {
	Description string
	ExecStart   string
}

var unitTmpl = template.Must(template.New("unit").Parse(unitTemplate))

// RenderUnits renders the systemd unit file contents for an
// execution+beacon client pair wired up on a given network. Pure string
// rendering: no file is written and no executor.Executor is touched — that
// is Task 4's job. Naming/writing the units as
// valve-node-exec.service / valve-node-beacon.service is also Task 4's job.
func RenderUnits(w WireConfig) (execUnit, beaconUnit string, err error) {
	net, ok := NetworkByChainID(w.ChainID)
	if !ok {
		return "", "", fmt.Errorf("catalog: unknown chain id %d", w.ChainID)
	}
	execClient, ok := ClientByID(w.ExecID)
	if !ok {
		return "", "", fmt.Errorf("catalog: unknown exec client %q", w.ExecID)
	}
	if execClient.Kind != "exec" {
		return "", "", fmt.Errorf("catalog: client %q is a %s client, not exec", w.ExecID, execClient.Kind)
	}
	beaconClient, ok := ClientByID(w.BeaconID)
	if !ok {
		return "", "", fmt.Errorf("catalog: unknown beacon client %q", w.BeaconID)
	}
	if beaconClient.Kind != "beacon" {
		return "", "", fmt.Errorf("catalog: client %q is a %s client, not beacon", w.BeaconID, beaconClient.Kind)
	}
	if !contains(net.ExecClients, w.ExecID) {
		return "", "", fmt.Errorf("catalog: %s is not a valid execution client on chain %d (%s)", w.ExecID, net.ChainID, net.Name)
	}
	if !contains(net.BeaconClients, w.BeaconID) {
		return "", "", fmt.Errorf("catalog: %s is not a valid beacon client on chain %d (%s)", w.BeaconID, net.ChainID, net.Name)
	}

	jwtPath := w.JWTPath
	if jwtPath == "" {
		jwtPath = path.Join(w.DataDir, "jwt.hex")
	}
	w.JWTPath = jwtPath

	execCmd, err := execCommand(w)
	if err != nil {
		return "", "", err
	}
	beaconCmd, err := beaconCommand(w, net)
	if err != nil {
		return "", "", err
	}

	execUnit, err = renderUnit("execution client ("+w.ExecID+")", execCmd)
	if err != nil {
		return "", "", err
	}
	beaconUnit, err = renderUnit("beacon client ("+w.BeaconID+")", beaconCmd)
	if err != nil {
		return "", "", err
	}
	return execUnit, beaconUnit, nil
}

func renderUnit(description, execStart string) (string, error) {
	var buf bytes.Buffer
	if err := unitTmpl.Execute(&buf, unitVars{Description: description, ExecStart: execStart}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// execCommand builds the ExecStart line for the execution-client unit.
// Flag sets are per task-3-brief.md Step 3, which is authoritative for
// this task (steps.ts does not render full exec-client CLI invocations).
// Where the brief and the learn data are both silent on a flag (the
// go-pulse/erigon-pulse network-selector, the go-pulse/geth binary name),
// see the data-provenance notes in task-3-report.md — nothing here is a
// blind guess beyond what the brief itself specifies.
func execCommand(w WireConfig) (string, error) {
	switch w.ExecID {
	case "reth":
		chain, ok := rethChainName[w.ChainID]
		if !ok {
			return "", fmt.Errorf("catalog: no reth --chain mapping for chain id %d", w.ChainID)
		}
		cmd := fmt.Sprintf(
			"reth node --chain %s --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr 127.0.0.1 --http.port %s",
			chain, w.DataDir, w.JWTPath, execHTTPPort,
		)
		if !w.Archive {
			cmd += " --full"
		}
		return cmd, nil

	case "geth":
		cmd := fmt.Sprintf(
			"geth --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr 127.0.0.1 --http.port %s",
			w.DataDir, w.JWTPath, execHTTPPort,
		)
		if w.Archive {
			cmd += " --gcmode archive"
		}
		return cmd, nil

	case "go-pulse":
		// go-pulse is the Geth-derived PulseChain execution client
		// (clients.ts): geth's flag surface, but installed at
		// /usr/local/bin/go-pulse (task-4b BuildCmd), so it's invoked by
		// that name, not geth's.
		cmd := fmt.Sprintf(
			"go-pulse --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr 127.0.0.1 --http.port %s",
			w.DataDir, w.JWTPath, execHTTPPort,
		)
		switch w.ChainID {
		case 369:
			cmd += " --pulsechain"
		case 943:
			// Verified E2E against the built go-pulse binary's --help on a
			// live box: --pulsechain.testnet is rejected ("flag provided
			// but not defined: -pulsechain.testnet"); the real selector is
			// --pulsechain-testnet-v4.
			cmd += " --pulsechain-testnet-v4"
		default:
			return "", fmt.Errorf("catalog: go-pulse is not valid on chain id %d", w.ChainID)
		}
		if w.Archive {
			cmd += " --gcmode archive"
		}
		return cmd, nil

	case "erigon-pulse":
		chain, ok := rethChainName[w.ChainID]
		if !ok {
			return "", fmt.Errorf("catalog: no erigon-pulse --chain mapping for chain id %d", w.ChainID)
		}
		cmd := fmt.Sprintf(
			"erigon-pulse --chain %s --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr 127.0.0.1 --http.port %s",
			chain, w.DataDir, w.JWTPath, execHTTPPort,
		)
		// erigon defaults to archive mode; pruning is opt-in via --prune flags.
		// erigon-2 full-node convention is --prune=hrtc. A wrong flag fails fast
		// at unit start, surfaced by the setup handshake.
		if !w.Archive {
			cmd += " --prune=hrtc"
		}
		return cmd, nil

	default:
		return "", fmt.Errorf("catalog: no exec unit template for client %q", w.ExecID)
	}
}

// beaconCommand builds the ExecStart line for the beacon-client unit.
// Flag sets are per task-3-brief.md Step 3 (lighthouse family verbatim)
// plus the checkpoint/genesis flags shown in steps.ts's "configure" step.
func beaconCommand(w WireConfig, net Network) (string, error) {
	switch w.BeaconID {
	case "lighthouse-pulse", "lighthouse":
		// Both lighthouse-family clients share their flag surface, but
		// lighthouse-pulse installs to /usr/local/bin/lighthouse-pulse
		// (task-4b BuildCmd) while upstream sigp lighthouse installs to
		// /usr/local/bin/lighthouse — invoke each by its own binary name.
		network, ok := lighthouseNetworkName[w.ChainID]
		if !ok {
			return "", fmt.Errorf("catalog: no lighthouse --network mapping for chain id %d", w.ChainID)
		}
		cmd := fmt.Sprintf(
			"%s bn --network %s --datadir %s --execution-endpoint %s --execution-jwt %s --checkpoint-sync-url %s --genesis-beacon-api-url %s --http --http-address 127.0.0.1 --http-port %s",
			w.BeaconID, network, w.DataDir, engineEndpoint, w.JWTPath, net.CheckpointURL, net.CheckpointURL, beaconHTTPPort,
		)
		return cmd, nil

	case "prysm-pulse":
		// prysm-pulse installs to /usr/local/bin/prysm-pulse (task-4b
		// BuildCmd), not the upstream beacon-chain binary name.
		cmd := fmt.Sprintf(
			"prysm-pulse --datadir=%s --execution-endpoint=%s --jwt-secret=%s --checkpoint-sync-url=%s --genesis-beacon-api-url=%s --rpc-host=127.0.0.1 --grpc-gateway-host=127.0.0.1 --grpc-gateway-port=%s",
			w.DataDir, engineEndpoint, w.JWTPath, net.CheckpointURL, net.CheckpointURL, beaconHTTPPort,
		)
		switch w.ChainID {
		case 369:
			cmd += " --pulsechain"
		case 943:
			// Verified E2E against the built prysm-pulse binary's --help on
			// a live box: --pulsechain-testnet is rejected ("flag provided
			// but not defined: -pulsechain-testnet"); the real selector is
			// --pulsechain-testnet-v4.
			cmd += " --pulsechain-testnet-v4"
		default:
			return "", fmt.Errorf("catalog: prysm-pulse is not valid on chain id %d", w.ChainID)
		}
		return cmd, nil

	default:
		return "", fmt.Errorf("catalog: no beacon unit template for client %q", w.BeaconID)
	}
}

func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}
