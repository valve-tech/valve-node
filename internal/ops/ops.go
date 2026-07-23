// Package ops implements the day-2 operator actions valve-node exposes on
// top of a wired target: starting/stopping/restarting the exec/beacon
// systemd services, clearing a service's data for a fresh resync, disk
// usage/size estimates, endpoint reachability, and a live (never-mutating)
// firewall checklist. Every func here takes a context.Context and an
// executor.Executor first (local or SSH — this package never knows which),
// per the same architectural seam every other package in valve-node is
// built on.
package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// Unit names, matching internal/setup/steps.go and internal/monitor/monitor.go.
const (
	execUnitName   = "valve-node-exec.service"
	beaconUnitName = "valve-node-beacon.service"
)

// enginePort is the fixed, non-configurable engine-API port (JWT-authed,
// execution<->beacon), matching internal/catalog/units.go's engineEndpoint.
const enginePort = 8551

// unitFor maps the API's svc identifier ("exec"|"beacon") to its systemd
// unit name. The empty string / ok=false return signals an invalid svc.
func unitFor(svc string) (string, bool) {
	switch svc {
	case "exec":
		return execUnitName, true
	case "beacon":
		return beaconUnitName, true
	default:
		return "", false
	}
}

// ---------------------------------------------------------------------
// ServiceAction
// ---------------------------------------------------------------------

// ServiceAction runs `systemctl <action> <unit>` for svc ("exec"|"beacon"),
// then reads back `systemctl is-active` for that unit as the returned
// active bool. An unrecognized svc or action is rejected before anything
// is run on the target.
func ServiceAction(ctx context.Context, e executor.Executor, svc, action string) (bool, error) {
	unit, ok := unitFor(svc)
	if !ok {
		return false, fmt.Errorf("ops: invalid service %q (want \"exec\" or \"beacon\")", svc)
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return false, fmt.Errorf("ops: invalid action %q (want \"start\", \"stop\", or \"restart\")", action)
	}

	res, err := e.Run(ctx, fmt.Sprintf("systemctl %s %s", action, unit), nil)
	if err != nil {
		return false, fmt.Errorf("ops: systemctl %s %s: %w", action, unit, err)
	}
	if res.ExitCode != 0 {
		return false, fmt.Errorf("ops: systemctl %s %s failed (exit %d): %s", action, unit, res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	return isActive(ctx, e, unit)
}

func isActive(ctx context.Context, e executor.Executor, unit string) (bool, error) {
	res, err := e.Run(ctx, fmt.Sprintf("systemctl is-active %s", unit), nil)
	if err != nil {
		return false, fmt.Errorf("ops: systemctl is-active %s: %w", unit, err)
	}
	// is-active exits non-zero for any non-"active" state (inactive,
	// failed, ...) — that's a valid reading, not a probe failure, so only
	// the transport error above is checked; the state itself comes from
	// stdout.
	return strings.TrimSpace(res.Stdout) == "active", nil
}

// ---------------------------------------------------------------------
// ClearService
// ---------------------------------------------------------------------

// ClearService stops svc's unit, deletes ONLY that client's own data
// subtree(s) under w.DataDir (catalog.Client.DataSubdirs — never the whole
// DataDir, so the JWT secret and the sibling client's data survive), and
// starts the unit again for a fresh resync.
//
// Safety: every computed delete path is refused if it resolves to DataDir
// itself, "/", or empty (clearPaths). If the stop fails, nothing is
// deleted. If the delete fails (partially or fully), ClearService returns
// an error WITHOUT attempting to start the unit again — the unit is left
// stopped and the error says so, rather than risk starting a service on
// top of a half-deleted data directory.
func ClearService(ctx context.Context, e executor.Executor, w catalog.WireConfig, svc string) error {
	unit, ok := unitFor(svc)
	if !ok {
		return fmt.Errorf("ops: invalid service %q (want \"exec\" or \"beacon\")", svc)
	}

	var clientID string
	switch svc {
	case "exec":
		clientID = w.ExecID
	case "beacon":
		clientID = w.BeaconID
	}
	client, ok := catalog.ClientByID(clientID)
	if !ok {
		return fmt.Errorf("ops: clear: unknown client id %q for service %q", clientID, svc)
	}

	paths, err := clearPaths(w.DataDir, client.DataSubdirs)
	if err != nil {
		return fmt.Errorf("ops: clear: %w", err)
	}

	res, err := e.Run(ctx, fmt.Sprintf("systemctl stop %s", unit), nil)
	if err != nil {
		return fmt.Errorf("ops: clear: systemctl stop %s: %w", unit, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("ops: clear: systemctl stop %s failed (exit %d): %s", unit, res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = shQuote(p)
	}
	rmCmd := "rm -rf -- " + strings.Join(quoted, " ")
	res, err = e.Run(ctx, rmCmd, nil)
	if err != nil {
		return fmt.Errorf("ops: clear: delete %s data failed: %w (unit left stopped)", svc, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("ops: clear: delete %s data failed (exit %d): %s (unit left stopped)", svc, res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	res, err = e.Run(ctx, fmt.Sprintf("systemctl start %s", unit), nil)
	if err != nil {
		return fmt.Errorf("ops: clear: systemctl start %s: %w", unit, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("ops: clear: systemctl start %s failed (exit %d): %s", unit, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

// clearPaths computes the delete path(s) for a clear operation — exactly
// DataDir + "/" + each of subdirs — and refuses (returning an error rather
// than any path) if DataDir is empty, if DataDir itself resolves to "/", if
// any subdir is empty, or if any computed path resolves (via path.Clean, so
// a trailing slash or a "." doesn't slip through) to DataDir itself or "/".
func clearPaths(dataDir string, subdirs []string) ([]string, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("refusing to clear: DataDir is empty")
	}
	if len(subdirs) == 0 {
		return nil, fmt.Errorf("refusing to clear: no known data subdirectories for this client")
	}
	cleanDataDir := path.Clean(dataDir)
	if cleanDataDir == "/" || cleanDataDir == "." {
		return nil, fmt.Errorf("refusing to clear: DataDir %q resolves to %q", dataDir, cleanDataDir)
	}

	paths := make([]string, 0, len(subdirs))
	for _, sub := range subdirs {
		if sub == "" {
			return nil, fmt.Errorf("refusing to clear: empty data subdir for DataDir %q", dataDir)
		}
		p := dataDir + "/" + sub
		cp := path.Clean(p)
		if cp == cleanDataDir || cp == "/" || cp == "" {
			return nil, fmt.Errorf("refusing to clear: computed delete path %q is unsafe", p)
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// ---------------------------------------------------------------------
// DiskUsage
// ---------------------------------------------------------------------

// DU is one point-in-time disk-usage/size-estimate reading for a target.
type DU struct {
	ExecBytes     uint64
	BeaconBytes   uint64
	DiskFreeBytes uint64

	// ExpectedExecBytes/ExpectedBeaconBytes are size ESTIMATES, not live
	// measurements — they vary by client and pruning, same caveat as
	// learn.valve.city's numbers they're ported from.
	ExpectedExecBytes   uint64
	ExpectedBeaconBytes uint64

	SyncLabel        string
	GenesisSyncLabel string
}

// beaconExpectedBytes is a rough, chain-independent estimate of a synced
// beacon chain's on-disk footprint after pruning. Unlike execution-client
// archive/full sizes (catalog.ExpectedBytes, ported verbatim from
// learn.valve.city's per-chain snapshot table), there is no per-chain
// figure for beacon-chain size in the catalog — it's dominated by slot
// count/pruning policy rather than chain-specific archive tiering, so a
// single coarse constant stands in until real per-chain data exists. DU's
// figures are always presented to the operator as estimates.
const beaconExpectedBytes uint64 = 250_000_000_000 // ~250GB

// DiskUsage reports current on-disk size of each client's own data (per
// catalog.Client.DataSubdirs, matching what ClearService deletes),
// filesystem free space, and the chain's expected dataset size / sync-time
// labels from the catalog.
func DiskUsage(ctx context.Context, e executor.Executor, w catalog.WireConfig) (DU, error) {
	var du DU

	net, ok := catalog.NetworkByChainID(w.ChainID)
	if !ok {
		return du, fmt.Errorf("ops: du: unknown chain id %d", w.ChainID)
	}
	du.SyncLabel = net.SyncLabel
	du.GenesisSyncLabel = net.GenesisSyncLabel

	expectedExec, err := catalog.ExpectedBytes(w.ChainID, w.Archive)
	if err != nil {
		return du, fmt.Errorf("ops: du: %w", err)
	}
	du.ExpectedExecBytes = expectedExec
	du.ExpectedBeaconBytes = beaconExpectedBytes

	execClient, ok := catalog.ClientByID(w.ExecID)
	if !ok {
		return du, fmt.Errorf("ops: du: unknown exec client id %q", w.ExecID)
	}
	beaconClient, ok := catalog.ClientByID(w.BeaconID)
	if !ok {
		return du, fmt.Errorf("ops: du: unknown beacon client id %q", w.BeaconID)
	}

	execBytes, err := duBytes(ctx, e, w.DataDir, execClient.DataSubdirs)
	if err != nil {
		return du, fmt.Errorf("ops: du: exec: %w", err)
	}
	du.ExecBytes = execBytes

	beaconBytes, err := duBytes(ctx, e, w.DataDir, beaconClient.DataSubdirs)
	if err != nil {
		return du, fmt.Errorf("ops: du: beacon: %w", err)
	}
	du.BeaconBytes = beaconBytes

	free, err := diskFreeBytes(ctx, e, w.DataDir)
	if err != nil {
		return du, fmt.Errorf("ops: du: %w", err)
	}
	du.DiskFreeBytes = free

	return du, nil
}

// duBytes sums `du -sb` across dataDir+"/"+subdir for each subdir. A
// subdir that doesn't exist yet (client never started, or only one of
// exec/beacon has synced any data) makes `du` report an error for that one
// argument on stderr and exit non-zero overall, while still printing sizes
// for the rest — so the exit code is ignored here and only lines that
// parse as "<bytes>\t<path>" are summed; a subdir with no such line
// contributes zero.
func duBytes(ctx context.Context, e executor.Executor, dataDir string, subdirs []string) (uint64, error) {
	if len(subdirs) == 0 {
		return 0, nil
	}
	paths := make([]string, len(subdirs))
	for i, s := range subdirs {
		paths[i] = shQuote(dataDir + "/" + s)
	}
	cmd := fmt.Sprintf("du -sb %s 2>/dev/null", strings.Join(paths, " "))
	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return 0, fmt.Errorf("du: %w", err)
	}

	var total uint64
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		n, perr := strconv.ParseUint(fields[0], 10, 64)
		if perr != nil {
			continue
		}
		total += n
	}
	return total, nil
}

// diskFreeBytes reports free bytes on the filesystem holding dataDir, via
// the same nearest-existing-ancestor `df` walk internal/setup's preflight
// check uses (a target's DataDir always exists by the time DiskUsage is a
// meaningful call — post-wire — but walking up is harmless and keeps this
// probe correct even called early).
func diskFreeBytes(ctx context.Context, e executor.Executor, dataDir string) (uint64, error) {
	cmd := fmt.Sprintf(
		`d=%s; while [ ! -d "$d" ]; do d=$(dirname "$d"); done; df -B1 --output=avail "$d" | tail -1`,
		shQuote(dataDir),
	)
	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return 0, fmt.Errorf("df: %w", err)
	}
	if res.ExitCode != 0 {
		return 0, fmt.Errorf("df failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if n, perr := strconv.ParseUint(line, 10, 64); perr == nil {
			return n, nil
		}
	}
	return 0, fmt.Errorf("could not parse df output: %q", res.Stdout)
}

// ---------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------

// EndpointInfo reports the local RPC URLs a target's clients are wired to,
// whether they answer right now, and (SSH targets only) a ready-to-copy
// tunnel command. Named EndpointInfo (not Endpoints) only because Go
// disallows a type and a func sharing a name in the same package — the
// Endpoints() func below is this type's sole constructor.
type EndpointInfo struct {
	ExecHTTP   string
	BeaconHTTP string

	ExecReachable   bool
	BeaconReachable bool
	ChainIDMatches  bool // exec's eth_chainId response matches w.ChainID

	Access     string // "local" | "ssh"
	TunnelHint string // "" for local targets
}

// Endpoints probes exec's eth_chainId and beacon's /eth/v1/node/version
// over the Executor (i.e. from ON the target box, exactly like the setup
// handshake does) — these URLs are loopback-bound and unreachable from the
// app host in SSH mode, so reachability can only ever be checked on-box.
func Endpoints(ctx context.Context, e executor.Executor, w catalog.WireConfig, sshMode bool, sshHostHint string) (EndpointInfo, error) {
	var ep EndpointInfo
	ep.ExecHTTP = fmt.Sprintf("http://127.0.0.1:%d", w.ExecHTTP())
	ep.BeaconHTTP = fmt.Sprintf("http://127.0.0.1:%d", w.BeaconHTTP())

	if sshMode {
		ep.Access = "ssh"
		ep.TunnelHint = fmt.Sprintf(
			"ssh -L %d:127.0.0.1:%d -L %d:127.0.0.1:%d root@%s",
			w.ExecHTTP(), w.ExecHTTP(), w.BeaconHTTP(), w.BeaconHTTP(), sshHostHint,
		)
	} else {
		ep.Access = "local"
	}

	res, err := e.Run(ctx,
		`curl -s -m 5 -X POST -H 'Content-Type: application/json' `+
			`--data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' `+
			ep.ExecHTTP, nil)
	if err == nil && res.ExitCode == 0 {
		if id, ok := parseHexResult(res.Stdout); ok {
			ep.ExecReachable = true
			ep.ChainIDMatches = id == uint64(w.ChainID)
		}
	}

	res, err = e.Run(ctx, "curl -s -m 5 -o /dev/null -w '%{http_code}' "+ep.BeaconHTTP+"/eth/v1/node/version", nil)
	if err == nil && res.ExitCode == 0 && strings.TrimSpace(res.Stdout) == "200" {
		ep.BeaconReachable = true
	}

	return ep, nil
}

// parseHexResult parses a JSON-RPC response whose "result" is a 0x-hex
// string, e.g. eth_chainId's. Duplicated from internal/monitor (which has
// its own copy for the same reason internal/setup and internal/monitor
// each keep their own shQuote — no shared util package exists for this
// yet, and importing monitor from ops just to reach one unexported helper
// isn't worth the coupling).
func parseHexResult(body string) (uint64, bool) {
	var resp struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return 0, false
	}
	hexStr := strings.TrimPrefix(resp.Result, "0x")
	if hexStr == "" {
		return 0, true
	}
	n, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ---------------------------------------------------------------------
// FirewallChecklist
// ---------------------------------------------------------------------

// CheckItem is one firewall checklist entry.
type CheckItem struct {
	ID     string
	Title  string
	Why    string
	Status string // "pass" | "fail" | "warn" | "unknown"
	Detail string
	Fix    string // copy-paste command(s); "" when Status is "pass"
}

// FirewallChecklist runs the live probes behind the v0.2 spec §5 checklist
// and returns their current status. It NEVER runs a mutating command (no
// `ufw allow`/`ufw enable`/`iptables -A` ever appears in a command this
// func passes to Executor.Run) — every suggested fix is returned as text
// in a CheckItem's Fix field for the operator to review and run themselves.
func FirewallChecklist(ctx context.Context, e executor.Executor, w catalog.WireConfig) ([]CheckItem, error) {
	beaconPorts, err := beaconP2PPorts(w.BeaconID)
	if err != nil {
		return nil, fmt.Errorf("ops: firewall: %w", err)
	}

	tcpRes, tcpErr := e.Run(ctx, "ss -ltn", nil)
	udpRes, udpErr := e.Run(ctx, "ss -lun", nil)
	if tcpErr != nil {
		return nil, fmt.Errorf("ops: firewall: ss -ltn: %w", tcpErr)
	}
	if udpErr != nil {
		return nil, fmt.Errorf("ops: firewall: ss -lun: %w", udpErr)
	}
	tcp, udp := tcpRes.Stdout, udpRes.Stdout

	items := []CheckItem{
		p2pOpenItem("exec-p2p-open", "Execution p2p port reachable", w.ExecP2P(), w.ExecP2P(), tcp, udp),
		p2pOpenItem("beacon-p2p-open", "Beacon p2p port reachable", beaconPorts.TCP, beaconPorts.UDP, tcp, udp),
		rpcNotPublicItem(w, tcp),
		firewallActiveItem(ctx, e, w, beaconPorts),
		sshAllowedItem(tcp),
	}
	return items, nil
}

// beaconP2P is a beacon client family's fixed (non-configurable, per the
// v0.2 spec decision) p2p listening ports.
type beaconP2P struct {
	TCP, UDP int
}

// beaconP2PPorts returns beaconID's p2p ports, per the plan's fixed
// defaults per client family: lighthouse family 9000/9000
// (tcp/udp), prysm family 13000/12000 (tcp/udp).
func beaconP2PPorts(beaconID string) (beaconP2P, error) {
	switch beaconID {
	case "lighthouse", "lighthouse-pulse":
		return beaconP2P{TCP: 9000, UDP: 9000}, nil
	case "prysm-pulse":
		return beaconP2P{TCP: 13000, UDP: 12000}, nil
	default:
		return beaconP2P{}, fmt.Errorf("no known p2p ports for beacon client %q", beaconID)
	}
}

// bindState classifies how a port is bound based on `ss -ltn`/`ss -lun`
// output: "wide" (0.0.0.0, *, or [::] — reachable from outside this box),
// "loopback" (127.0.0.1 or ::1 — local only), or "" if the port isn't
// listening at all in this output.
func bindState(ssOutput string, port int) string {
	suffix := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(ssOutput, "\n") {
		for _, f := range strings.Fields(line) {
			if !strings.HasSuffix(f, suffix) {
				continue
			}
			addr := strings.TrimSuffix(f, suffix)
			switch addr {
			case "127.0.0.1", "::1", "[::1]":
				return "loopback"
			default:
				return "wide"
			}
		}
	}
	return ""
}

func p2pOpenItem(id, title string, tcpPort, udpPort int, tcp, udp string) CheckItem {
	tcpState := bindState(tcp, tcpPort)
	udpState := bindState(udp, udpPort)

	why := "Peers need to reach this port from the internet to connect inbound; " +
		"a loopback-only or closed p2p port means slower sync and no inbound peers."
	fix := fmt.Sprintf("ufw allow %d/tcp\nufw allow %d/udp", tcpPort, udpPort)

	switch {
	case tcpState == "wide" || udpState == "wide":
		return CheckItem{ID: id, Title: title, Why: why, Status: "pass",
			Detail: fmt.Sprintf("listening on a publicly-reachable address (tcp %d, udp %d)", tcpPort, udpPort)}
	case tcpState == "loopback" || udpState == "loopback":
		return CheckItem{ID: id, Title: title, Why: why, Status: "warn",
			Detail: "peers can't reach you — sync will be slower/inbound-less", Fix: fix}
	default:
		return CheckItem{ID: id, Title: title, Why: why, Status: "unknown",
			Detail: fmt.Sprintf("not currently listening on tcp %d / udp %d (service may not be up yet)", tcpPort, udpPort)}
	}
}

func rpcNotPublicItem(w catalog.WireConfig, tcp string) CheckItem {
	type namedPort struct {
		name string
		port int
	}
	ports := []namedPort{
		{"exec HTTP", w.ExecHTTP()},
		{"engine API", enginePort},
		{"beacon HTTP", w.BeaconHTTP()},
	}

	why := "Exec HTTP, the engine API, and beacon HTTP carry unauthenticated wallet/node control surface " +
		"(exec HTTP included) — binding any of them to a public address lets anyone on the internet reach it."

	var wide []string
	for _, p := range ports {
		if bindState(tcp, p.port) == "wide" {
			wide = append(wide, fmt.Sprintf("%s (%d)", p.name, p.port))
		}
	}
	sort.Strings(wide)

	if len(wide) > 0 {
		return CheckItem{
			ID: "rpc-not-public", Title: "RPC/engine ports not publicly exposed", Why: why,
			Status: "fail",
			Detail: fmt.Sprintf("bound to a public address: %s — anyone on the internet can reach it", strings.Join(wide, ", ")),
			Fix:    "bind the affected flag(s) back to 127.0.0.1 in the client's unit and restart the service",
		}
	}
	return CheckItem{
		ID: "rpc-not-public", Title: "RPC/engine ports not publicly exposed", Why: why,
		Status: "pass",
		Detail: "exec HTTP, engine API, and beacon HTTP are all loopback-bound (or not listening)",
	}
}

func firewallActiveItem(ctx context.Context, e executor.Executor, w catalog.WireConfig, beaconPorts beaconP2P) CheckItem {
	why := "With no firewall active, every service any client binds to a public address (or misconfigures) " +
		"is reachable from the internet with no additional layer of defense."

	res, err := e.Run(ctx, "ufw status", nil)
	if err == nil && res.ExitCode == 0 && strings.Contains(strings.ToLower(res.Stdout), "status: active") {
		return CheckItem{ID: "firewall-active", Title: "Firewall active", Why: why,
			Status: "pass", Detail: strings.TrimSpace(res.Stdout)}
	}

	detail := "no active firewall detected"
	if err == nil {
		out := strings.TrimSpace(res.Stdout)
		if out != "" {
			detail = fmt.Sprintf("no active firewall detected (%s)", out)
		}
	}

	// Ready-to-paste block. FIRST line always allows SSH — never hand the
	// operator a lockout by suggesting `ufw enable` before their own access
	// is preserved.
	fix := strings.Join([]string{
		"ufw allow 22/tcp",
		fmt.Sprintf("ufw allow %d/tcp", w.ExecP2P()),
		fmt.Sprintf("ufw allow %d/udp", w.ExecP2P()),
		fmt.Sprintf("ufw allow %d/tcp", beaconPorts.TCP),
		fmt.Sprintf("ufw allow %d/udp", beaconPorts.UDP),
		"ufw default deny incoming",
		"ufw enable",
	}, "\n")

	return CheckItem{ID: "firewall-active", Title: "Firewall active", Why: why,
		Status: "warn", Detail: detail, Fix: fix}
}

func sshAllowedItem(tcp string) CheckItem {
	why := "Before suggesting any firewall-enable command, confirm SSH is actually listening — " +
		"enabling a firewall that doesn't already allow your own access is how you get locked out."

	if bindState(tcp, 22) != "" {
		return CheckItem{ID: "ssh-allowed", Title: "SSH still allowed", Why: why,
			Status: "pass", Detail: "sshd is listening on 22"}
	}
	return CheckItem{ID: "ssh-allowed", Title: "SSH still allowed", Why: why,
		Status: "warn",
		Detail: "sshd does not appear to be listening on 22 — verify SSH access before enabling any firewall"}
}

// ---------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------

// shQuote single-quotes s for safe interpolation into a `sh -c` command
// string, escaping any embedded single quotes. Duplicated from
// internal/setup/steps.go and internal/monitor/monitor.go, each of which
// keeps its own copy rather than share one via the executor package — see
// steps.go's comment; ops follows the same established convention.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
