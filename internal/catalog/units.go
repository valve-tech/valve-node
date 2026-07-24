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

	// ExecHTTPPort, BeaconHTTPPort, and ExecP2PPort are the ports each
	// client is wired to. Zero value means "use the default" — resolved
	// via the ExecHTTP/BeaconHTTP/ExecP2P methods below, never read
	// directly, so config-file backward compat (existing configs with no
	// port fields at all) keeps working unchanged.
	ExecHTTPPort   int // default 8545
	BeaconHTTPPort int // default 5052
	ExecP2PPort    int // default 30303

	// RPCBindAddr is the host address the execution and beacon HTTP RPC
	// endpoints bind to. Empty means the default (127.0.0.1, loopback-only)
	// — resolved via RPCBind(), never read directly, so existing configs
	// keep binding to loopback unchanged. Set it to a routable address
	// (e.g. the node's Tailscale IP) to reach the RPC from another machine.
	// The engine API (authrpc, 8551) and prysm's native rpc-host always
	// stay loopback regardless — they are never exposed.
	RPCBindAddr string
}

// Default ports, applied whenever the corresponding WireConfig field is
// left at its zero value.
const (
	defaultExecHTTPPort   = 8545
	defaultBeaconHTTPPort = 5052
	defaultExecP2PPort    = 30303
)

// ExecHTTP returns the execution client's HTTP RPC port, resolving the
// zero value to the default (8545).
func (w WireConfig) ExecHTTP() int {
	if w.ExecHTTPPort == 0 {
		return defaultExecHTTPPort
	}
	return w.ExecHTTPPort
}

// BeaconHTTP returns the beacon client's HTTP API port, resolving the zero
// value to the default (5052).
func (w WireConfig) BeaconHTTP() int {
	if w.BeaconHTTPPort == 0 {
		return defaultBeaconHTTPPort
	}
	return w.BeaconHTTPPort
}

// ExecP2P returns the execution client's devp2p listening port, resolving
// the zero value to the default (30303).
func (w WireConfig) ExecP2P() int {
	if w.ExecP2PPort == 0 {
		return defaultExecP2PPort
	}
	return w.ExecP2PPort
}

// RPCBind returns the host address the exec/beacon HTTP RPC binds to,
// resolving the empty value to the loopback default (127.0.0.1). This is
// also the address valve-node's own on-box probes (monitor, diagnostics,
// setup handshake) must target — a client bound to a single non-loopback
// address no longer answers on 127.0.0.1.
func (w WireConfig) RPCBind() string {
	if w.RPCBindAddr == "" {
		return "127.0.0.1"
	}
	return w.RPCBindAddr
}

// engineEndpoint is the local engine-API (JSON-RPC over HTTP, JWT-authed)
// endpoint the beacon client talks to the execution client on. Fixed by
// the plan decision — always loopback, always 8551.
const engineEndpoint = "http://127.0.0.1:8551"

// ServiceUser/ServiceGroup are the dedicated unprivileged system account
// the execution and beacon services run as (the User=/Group= lines in the
// rendered units). Setup's account step creates the account; the wire step
// chowns the data dir to it.
const (
	ServiceUser  = "valve-node"
	ServiceGroup = "valve-node"
)

const unitTemplate = `[Unit]
Description=valve-node {{.Description}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={{.User}}
Group={{.Group}}
ExecStart={{.ExecStart}}
Restart=always
RestartSec=5
LimitNOFILE=1048576
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths={{.DataDir}}
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictSUIDSGID=true
{{- if .NetBindCap}}
AmbientCapabilities=CAP_NET_BIND_SERVICE{{end}}

[Install]
WantedBy=multi-user.target
`

type unitVars struct {
	Description string
	ExecStart   string
	User, Group string
	DataDir     string
	// NetBindCap grants CAP_NET_BIND_SERVICE when one of this unit's
	// configured ports is privileged (<1024) — without it the
	// unprivileged service user cannot bind such a port.
	NetBindCap bool
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

	execUnit, err = renderUnit("execution client ("+w.ExecID+")", execCmd, w,
		w.ExecHTTP() < 1024 || w.ExecP2P() < 1024)
	if err != nil {
		return "", "", err
	}
	beaconUnit, err = renderUnit("beacon client ("+w.BeaconID+")", beaconCmd, w,
		w.BeaconHTTP() < 1024)
	if err != nil {
		return "", "", err
	}
	return execUnit, beaconUnit, nil
}

func renderUnit(description, execStart string, w WireConfig, netBindCap bool) (string, error) {
	var buf bytes.Buffer
	err := unitTmpl.Execute(&buf, unitVars{
		Description: description,
		ExecStart:   execStart,
		User:        ServiceUser,
		Group:       ServiceGroup,
		DataDir:     w.DataDir,
		NetBindCap:  netBindCap,
	})
	if err != nil {
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
			"reth node --chain %s --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr %s --http.port %d --port %d",
			chain, w.DataDir, w.JWTPath, w.RPCBind(), w.ExecHTTP(), w.ExecP2P(),
		)
		if !w.Archive {
			cmd += " --full"
		}
		return cmd, nil

	case "geth":
		cmd := fmt.Sprintf(
			"geth --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr %s --http.port %d --port %d",
			w.DataDir, w.JWTPath, w.RPCBind(), w.ExecHTTP(), w.ExecP2P(),
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
			"go-pulse --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr %s --http.port %d --port %d",
			w.DataDir, w.JWTPath, w.RPCBind(), w.ExecHTTP(), w.ExecP2P(),
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
			"erigon-pulse --chain %s --datadir %s --authrpc.jwtsecret %s --authrpc.addr 127.0.0.1 --authrpc.port 8551 --http --http.addr %s --http.port %d --port %d",
			chain, w.DataDir, w.JWTPath, w.RPCBind(), w.ExecHTTP(), w.ExecP2P(),
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
		datadir := path.Join(w.DataDir, "beacon")
		cmd := fmt.Sprintf(
			"%s bn --network %s --datadir %s --execution-endpoint %s --execution-jwt %s --checkpoint-sync-url %s --genesis-beacon-api-url %s --http --http-address %s --http-port %d",
			w.BeaconID, network, datadir, engineEndpoint, w.JWTPath, net.CheckpointURL, net.CheckpointURL, w.RPCBind(), w.BeaconHTTP(),
		)
		return cmd, nil

	case "prysm-pulse":
		// prysm-pulse installs to /usr/local/bin/prysm-pulse (task-4b
		// BuildCmd), not the upstream beacon-chain binary name.
		datadir := path.Join(w.DataDir, "beacondata")
		cmd := fmt.Sprintf(
			"prysm-pulse --datadir=%s --execution-endpoint=%s --jwt-secret=%s --checkpoint-sync-url=%s --genesis-beacon-api-url=%s --rpc-host=127.0.0.1 --grpc-gateway-host=%s --grpc-gateway-port=%d",
			datadir, engineEndpoint, w.JWTPath, net.CheckpointURL, net.CheckpointURL, w.RPCBind(), w.BeaconHTTP(),
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
