package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
	"github.com/valve-tech/valve-node/internal/logwatch"
)

// DiagnoseOpts carries the app-host-side context NetworkDiagnostics needs
// beyond the executor: SSH mode/host for the inbound dial probe (the one
// probe that runs from the app host rather than on the target), and an
// injectable dialer so tests never open real sockets.
type DiagnoseOpts struct {
	SSHMode bool
	SSHHost string // bare host/IP (no user@); "" skips the inbound probe
	// Dial dials network/addr with a timeout; nil means net.DialTimeout.
	Dial func(network, addr string, timeout time.Duration) error
}

// inboundDialTimeout bounds each app-host → target p2p dial. Short on
// purpose: a healthy open port answers a TCP handshake near-instantly, and
// the whole diagnostics run should stay interactive.
const inboundDialTimeout = 3 * time.Second

// NetworkDiagnostics runs the network-stack troubleshooting ladder against
// the target: services → local RPC/API → p2p listeners → inbound
// reachability (SSH mode) → outbound connectivity → peers → sync → journal
// error signatures. It is a LADDER, not a survey: checks run in order and
// the suite stops at the first "fail" — the returned slice ends at the
// failing rung, which is the root cause the operator should act on
// ("check, check, check, failed here"). Warns don't stop the ladder. Like
// FirewallChecklist, it NEVER runs a mutating command — every remedy is
// copy-paste text in a CheckItem's Fix.
func NetworkDiagnostics(ctx context.Context, e executor.Executor, w catalog.WireConfig, opts DiagnoseOpts) ([]CheckItem, error) {
	network, ok := catalog.NetworkByChainID(w.ChainID)
	if !ok {
		return nil, fmt.Errorf("ops: diagnostics: unknown chain id %d", w.ChainID)
	}
	beaconPorts, err := beaconP2PPorts(w.BeaconID)
	if err != nil {
		return nil, fmt.Errorf("ops: diagnostics: %w", err)
	}

	var items []CheckItem
	// add appends the item and reports whether the ladder may continue —
	// false at the first "fail", so no later (now-pointless) probe runs.
	add := func(item CheckItem) bool {
		items = append(items, item)
		return item.Status != "fail"
	}

	execActive, beaconActive := unitActiveStates(ctx, e)
	if !add(servicesItem(execActive, beaconActive)) {
		return items, nil
	}
	// Past this rung both services are active — later probes can treat an
	// unanswered RPC/API as its own finding, not a service-down echo.
	if !add(execRPCItem(ctx, e, w)) {
		return items, nil
	}
	if !add(beaconAPIItem(ctx, e, w)) {
		return items, nil
	}

	tcpRes, tcpErr := e.Run(ctx, "ss -ltn", nil)
	udpRes, udpErr := e.Run(ctx, "ss -lun", nil)
	if tcpErr != nil {
		return nil, fmt.Errorf("ops: diagnostics: ss -ltn: %w", tcpErr)
	}
	if udpErr != nil {
		return nil, fmt.Errorf("ops: diagnostics: ss -lun: %w", udpErr)
	}
	// p2pOpenItem never fails (pass/warn/unknown), but keep the ladder
	// idiom uniform anyway.
	if !add(p2pOpenItem("exec-p2p-open", "Execution p2p port reachable", w.ExecP2P(), w.ExecP2P(), tcpRes.Stdout, udpRes.Stdout)) {
		return items, nil
	}
	if !add(p2pOpenItem("beacon-p2p-open", "Beacon p2p port reachable", beaconPorts.TCP, beaconPorts.UDP, tcpRes.Stdout, udpRes.Stdout)) {
		return items, nil
	}

	if opts.SSHMode && opts.SSHHost != "" {
		if !add(inboundItem(opts, w.ExecP2P(), beaconPorts.TCP)) {
			return items, nil
		}
	}

	if !add(outboundItem(ctx, e, network.CheckpointURL)) {
		return items, nil
	}
	if !add(execPeersItem(ctx, e, w)) {
		return items, nil
	}
	if !add(beaconPeersItem(ctx, e, w)) {
		return items, nil
	}
	if !add(syncItem(ctx, e, w)) {
		return items, nil
	}
	add(journalItem(ctx, e))
	return items, nil
}

// unitActiveStates reads `systemctl is-active` for both units. Like
// ownActiveUnitPorts in internal/setup, the exit code is ignored (is-active
// exits non-zero whenever any listed unit isn't active) and only the
// per-line output — one line per unit, in argument order — is read.
func unitActiveStates(ctx context.Context, e executor.Executor) (execActive, beaconActive bool) {
	res, err := e.Run(ctx, fmt.Sprintf("systemctl is-active %s %s", execUnitName, beaconUnitName), nil)
	if err != nil {
		return false, false
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "active" {
		execActive = true
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) == "active" {
		beaconActive = true
	}
	return execActive, beaconActive
}

func servicesItem(execActive, beaconActive bool) CheckItem {
	why := "Nothing else on this list can work until both the execution and beacon services are running — " +
		"start here whenever anything below fails."
	if execActive && beaconActive {
		return CheckItem{ID: "diag-services", Title: "Node services running", Why: why,
			Status: "pass", Detail: "both valve-node-exec and valve-node-beacon are active"}
	}

	var down, fixes []string
	if !execActive {
		down = append(down, execUnitName)
		fixes = append(fixes, "systemctl start "+execUnitName, fmt.Sprintf("journalctl -u %s -n 50 --no-pager", execUnitName))
	}
	if !beaconActive {
		down = append(down, beaconUnitName)
		fixes = append(fixes, "systemctl start "+beaconUnitName, fmt.Sprintf("journalctl -u %s -n 50 --no-pager", beaconUnitName))
	}
	return CheckItem{ID: "diag-services", Title: "Node services running", Why: why,
		Status: "fail",
		Detail: fmt.Sprintf("not active: %s", strings.Join(down, ", ")),
		Fix:    strings.Join(fixes, "\n"),
	}
}

func execRPCItem(ctx context.Context, e executor.Executor, w catalog.WireConfig) CheckItem {
	why := "The execution client's local JSON-RPC is how everything (the beacon client's engine calls aside) " +
		"talks to your node — if it doesn't answer on loopback, the client is down, still starting, or misconfigured."
	addr := fmt.Sprintf("http://%s:%d", w.RPCBind(), w.ExecHTTP())

	res, err := e.Run(ctx,
		`curl -s -m 5 -X POST -H 'Content-Type: application/json' `+
			`--data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' `+
			addr, nil)
	if err == nil && res.ExitCode == 0 {
		if id, ok := parseHexResult(res.Stdout); ok {
			if id == uint64(w.ChainID) {
				return CheckItem{ID: "diag-exec-rpc", Title: "Execution RPC answering", Why: why,
					Status: "pass", Detail: fmt.Sprintf("%s answered eth_chainId with the expected chain id %d", addr, w.ChainID)}
			}
			return CheckItem{ID: "diag-exec-rpc", Title: "Execution RPC answering", Why: why,
				Status: "fail",
				Detail: fmt.Sprintf("%s answered with chain id %d, but this target is configured for chain %d — the client is on the wrong network", addr, id, w.ChainID),
				Fix:    "re-run setup so the unit files match the configured chain, then clear + resync if the data dir holds another network's chain data",
			}
		}
	}

	return CheckItem{ID: "diag-exec-rpc", Title: "Execution RPC answering", Why: why,
		Status: "fail",
		Detail: fmt.Sprintf("the execution service is running but %s did not answer eth_chainId — likely still starting up (large chains can take minutes) or crashed mid-start", addr),
		Fix:    fmt.Sprintf("journalctl -u %s -n 50 --no-pager", execUnitName),
	}
}

func beaconAPIItem(ctx context.Context, e executor.Executor, w catalog.WireConfig) CheckItem {
	why := "The beacon client's local HTTP API is where sync and peer state come from — " +
		"if it doesn't answer on loopback, the client is down, still starting, or misconfigured."
	addr := fmt.Sprintf("http://%s:%d", w.RPCBind(), w.BeaconHTTP())

	res, err := e.Run(ctx, "curl -s -m 5 -o /dev/null -w '%{http_code}' "+addr+"/eth/v1/node/version", nil)
	if err == nil && res.ExitCode == 0 && strings.TrimSpace(res.Stdout) == "200" {
		return CheckItem{ID: "diag-beacon-api", Title: "Beacon API answering", Why: why,
			Status: "pass", Detail: addr + "/eth/v1/node/version returned 200"}
	}
	return CheckItem{ID: "diag-beacon-api", Title: "Beacon API answering", Why: why,
		Status: "fail",
		Detail: fmt.Sprintf("the beacon service is running but %s/eth/v1/node/version did not answer — likely still starting up or crashed mid-start", addr),
		Fix:    fmt.Sprintf("journalctl -u %s -n 50 --no-pager", beaconUnitName),
	}
}

func inboundItem(opts DiagnoseOpts, execP2P, beaconTCP int) CheckItem {
	why := "This is the only probe run from OUTSIDE the target (from the machine running valve-node): it dials " +
		"the target's public p2p TCP ports directly, so it catches hosting-provider firewalls and NAT that " +
		"an on-box listener check can't see. UDP can't be verified this way, so a pass here is necessary but " +
		"not sufficient for discovery."
	dial := opts.Dial
	if dial == nil {
		dial = func(network, addr string, timeout time.Duration) error {
			conn, err := net.DialTimeout(network, addr, timeout)
			if err != nil {
				return err
			}
			return conn.Close()
		}
	}

	var unreachable []string
	for _, port := range []int{execP2P, beaconTCP} {
		addr := net.JoinHostPort(opts.SSHHost, strconv.Itoa(port))
		if err := dial("tcp", addr, inboundDialTimeout); err != nil {
			unreachable = append(unreachable, fmt.Sprintf("%s (%v)", addr, err))
		}
	}

	if len(unreachable) == 0 {
		return CheckItem{ID: "diag-p2p-inbound", Title: "P2P ports reachable from outside", Why: why,
			Status: "pass",
			Detail: fmt.Sprintf("tcp %d and %d on %s accepted a connection from this machine", execP2P, beaconTCP, opts.SSHHost)}
	}
	return CheckItem{ID: "diag-p2p-inbound", Title: "P2P ports reachable from outside", Why: why,
		Status: "warn",
		Detail: "could not connect from this machine: " + strings.Join(unreachable, "; "),
		Fix: fmt.Sprintf("ufw allow %d/tcp\nufw allow %d/udp\nufw allow %d/tcp\nufw allow %d/udp\n"+
			"# also check your hosting provider's firewall / security-group rules for these ports",
			execP2P, execP2P, beaconTCP, beaconTCP),
	}
}

func outboundItem(ctx context.Context, e executor.Executor, checkpointURL string) CheckItem {
	why := "The node needs outbound DNS + HTTPS to reach its checkpoint-sync endpoint (and peers in general); " +
		"an egress-filtered or DNS-broken box will sit at 0 peers forever with no obvious local error."

	res, err := e.Run(ctx, fmt.Sprintf("curl -s -m 8 -o /dev/null %s", shQuote(checkpointURL)), nil)
	if err == nil && res.ExitCode == 0 {
		return CheckItem{ID: "diag-outbound", Title: "Outbound connectivity", Why: why,
			Status: "pass", Detail: fmt.Sprintf("the target reached %s (DNS, TLS, and outbound HTTPS all work)", checkpointURL)}
	}

	detail := fmt.Sprintf("could not reach %s from the target", checkpointURL)
	if err == nil {
		switch res.ExitCode {
		case 6:
			detail += " — DNS resolution failed"
		case 7:
			detail += " — connection refused or blocked"
		case 28:
			detail += " — timed out (egress filtered?)"
		default:
			detail += fmt.Sprintf(" (curl exit %d)", res.ExitCode)
		}
	}
	return CheckItem{ID: "diag-outbound", Title: "Outbound connectivity", Why: why,
		Status: "fail",
		Detail: detail,
		Fix: "check /etc/resolv.conf and the box's outbound firewall/egress rules; " +
			"verify with: curl -v " + checkpointURL,
	}
}

func execPeersItem(ctx context.Context, e executor.Executor, w catalog.WireConfig) CheckItem {
	why := "Peer count is the single clearest signal of p2p health: 0 peers means the node can't sync at all, " +
		"and a persistently low count usually means the p2p port isn't reachable from the internet."
	addr := fmt.Sprintf("http://%s:%d", w.RPCBind(), w.ExecHTTP())

	res, err := e.Run(ctx,
		`curl -s -m 5 -X POST -H 'Content-Type: application/json' `+
			`--data '{"jsonrpc":"2.0","id":1,"method":"net_peerCount","params":[]}' `+
			addr, nil)
	if err == nil && res.ExitCode == 0 {
		if n, ok := parseHexResult(res.Stdout); ok {
			return peersVerdict("diag-exec-peers", "Execution peers", why, int(n), 5,
				"peer discovery can take a few minutes after a (re)start; if the count stays at 0, "+
					"work through the p2p-port and inbound-reachability checks above")
		}
	}
	return CheckItem{ID: "diag-exec-peers", Title: "Execution peers", Why: why,
		Status: "unknown", Detail: "the execution RPC did not answer net_peerCount (see the RPC check)"}
}

func beaconPeersItem(ctx context.Context, e executor.Executor, w catalog.WireConfig) CheckItem {
	why := "The beacon client needs peers to follow the chain; 0 peers means it can't sync, and a low count " +
		"usually points at an unreachable p2p port."
	addr := fmt.Sprintf("http://%s:%d", w.RPCBind(), w.BeaconHTTP())

	res, err := e.Run(ctx, "curl -s -m 5 "+addr+"/eth/v1/node/peer_count", nil)
	if err == nil && res.ExitCode == 0 {
		if n, ok := parseBeaconPeerCount(res.Stdout); ok {
			return peersVerdict("diag-beacon-peers", "Beacon peers", why, n, 10,
				"peer discovery can take a few minutes after a (re)start; if the count stays at 0, "+
					"work through the p2p-port and inbound-reachability checks above")
		}
	}
	return CheckItem{ID: "diag-beacon-peers", Title: "Beacon peers", Why: why,
		Status: "unknown", Detail: "the beacon API did not answer /eth/v1/node/peer_count (see the API check)"}
}

// peersVerdict classifies a peer count: 0 → fail, 1..healthy-1 → warn,
// ≥healthy → pass. The healthy thresholds (5 exec / 10 beacon) are
// deliberately loose — this is a "something is wrong" detector, not a
// peering benchmark.
func peersVerdict(id, title, why string, count, healthy int, lowFix string) CheckItem {
	switch {
	case count == 0:
		return CheckItem{ID: id, Title: title, Why: why, Status: "fail",
			Detail: "0 peers — the client cannot sync in this state", Fix: lowFix}
	case count < healthy:
		return CheckItem{ID: id, Title: title, Why: why, Status: "warn",
			Detail: fmt.Sprintf("only %d peers connected (healthy is roughly %d+)", count, healthy), Fix: lowFix}
	default:
		return CheckItem{ID: id, Title: title, Why: why, Status: "pass",
			Detail: fmt.Sprintf("%d peers connected", count)}
	}
}

func syncItem(ctx context.Context, e executor.Executor, w catalog.WireConfig) CheckItem {
	why := "Whether each client considers itself in sync — a node that never leaves 'syncing' despite healthy " +
		"peers usually has a stalled counterpart (beacon waiting on exec or vice versa) or is simply mid-initial-sync."

	execAddr := fmt.Sprintf("http://%s:%d", w.RPCBind(), w.ExecHTTP())
	beaconAddr := fmt.Sprintf("http://%s:%d", w.RPCBind(), w.BeaconHTTP())

	execRes, execErr := e.Run(ctx,
		`curl -s -m 5 -X POST -H 'Content-Type: application/json' `+
			`--data '{"jsonrpc":"2.0","id":1,"method":"eth_syncing","params":[]}' `+
			execAddr, nil)
	beaconRes, beaconErr := e.Run(ctx, "curl -s -m 5 "+beaconAddr+"/eth/v1/node/syncing", nil)

	execSyncing, execOK := false, false
	if execErr == nil && execRes.ExitCode == 0 {
		execSyncing, execOK = parseEthSyncingResult(execRes.Stdout)
	}
	beaconSyncing, beaconDistance, beaconOK := false, uint64(0), false
	if beaconErr == nil && beaconRes.ExitCode == 0 {
		beaconSyncing, beaconDistance, beaconOK = parseBeaconSyncing(beaconRes.Stdout)
	}

	if !execOK || !beaconOK {
		return CheckItem{ID: "diag-sync", Title: "Sync status", Why: why,
			Status: "unknown", Detail: "could not read sync status from one or both clients (see the RPC/API checks)"}
	}

	if !execSyncing && !beaconSyncing {
		return CheckItem{ID: "diag-sync", Title: "Sync status", Why: why,
			Status: "pass", Detail: "both clients report they are in sync"}
	}

	var parts []string
	if execSyncing {
		parts = append(parts, "execution client is syncing")
	}
	if beaconSyncing {
		parts = append(parts, fmt.Sprintf("beacon client is syncing (%d slots behind)", beaconDistance))
	}
	return CheckItem{ID: "diag-sync", Title: "Sync status", Why: why,
		Status: "warn",
		Detail: strings.Join(parts, "; ") + " — normal during initial sync; investigate only if peers are healthy and the distance isn't shrinking over time",
	}
}

// journalScanLines is how much recent journal the signature scan reads —
// enough to catch a crash-restart loop's output without dragging hours of
// history into a point-in-time diagnosis.
const journalScanLines = 200

func journalItem(ctx context.Context, e executor.Executor) CheckItem {
	why := "Known failure signatures in the recent journal (auth errors, port clashes, stalls, low peers) " +
		"usually name the root cause directly when the probes above only show the symptom."

	res, err := e.Run(ctx, fmt.Sprintf(
		"journalctl -u %s -u %s -n %d --no-pager -o cat",
		execUnitName, beaconUnitName, journalScanLines), nil)
	if err != nil || res.ExitCode != 0 {
		return CheckItem{ID: "diag-journal", Title: "Journal error signatures", Why: why,
			Status: "unknown", Detail: "could not read the journal on the target"}
	}

	counts := map[string]int{}
	worst := logwatch.Hit{}
	now := time.Now()
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		hit, ok := logwatch.Classify("", line, now)
		if !ok || hit.Signature == "" {
			continue
		}
		counts[hit.Signature]++
		if severityRank(hit.Severity) > severityRank(worst.Severity) {
			worst = hit
		}
	}

	if len(counts) == 0 {
		return CheckItem{ID: "diag-journal", Title: "Journal error signatures", Why: why,
			Status: "pass", Detail: fmt.Sprintf("no known failure signatures in the last %d journal lines", journalScanLines)}
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	summary := make([]string, 0, len(names))
	for _, name := range names {
		summary = append(summary, fmt.Sprintf("%s ×%d", name, counts[name]))
	}
	detail := "matched: " + strings.Join(summary, ", ") + ". " + worst.Explain
	if worst.LearnURL != "" {
		detail += " (" + worst.LearnURL + ")"
	}

	status := "warn"
	if severityRank(worst.Severity) >= severityRank("error") {
		status = "fail"
	}
	return CheckItem{ID: "diag-journal", Title: "Journal error signatures", Why: why,
		Status: status,
		Detail: detail,
		Fix:    "open the Logs screen for the live tail and AI explanations of these lines",
	}
}

func severityRank(sev string) int {
	switch sev {
	case "critical":
		return 3
	case "error":
		return 2
	case "warn":
		return 1
	default:
		return 0
	}
}

// parseEthSyncingResult reads eth_syncing's result: JSON false means "in
// sync", any object means "syncing". ok=false when the body isn't a
// readable JSON-RPC response.
func parseEthSyncingResult(body string) (syncing, ok bool) {
	var resp struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || len(resp.Result) == 0 {
		return false, false
	}
	trimmed := strings.TrimSpace(string(resp.Result))
	switch {
	case trimmed == "false":
		return false, true
	case strings.HasPrefix(trimmed, "{"):
		return true, true
	default:
		return false, false
	}
}

// parseBeaconSyncing reads /eth/v1/node/syncing. The beacon API spec types
// head_slot/sync_distance as decimal strings; asUint tolerates bare
// numbers too, since clients differ.
func parseBeaconSyncing(body string) (syncing bool, distance uint64, ok bool) {
	var resp struct {
		Data struct {
			IsSyncing    bool `json:"is_syncing"`
			SyncDistance any  `json:"sync_distance"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return false, 0, false
	}
	distance, _ = asUint(resp.Data.SyncDistance)
	return resp.Data.IsSyncing, distance, true
}

// parseBeaconPeerCount reads /eth/v1/node/peer_count's data.connected
// (a decimal string per the beacon API spec; bare numbers tolerated).
func parseBeaconPeerCount(body string) (int, bool) {
	var resp struct {
		Data struct {
			Connected any `json:"connected"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Data.Connected == nil {
		return 0, false
	}
	n, ok := asUint(resp.Data.Connected)
	return int(n), ok
}

// asUint coerces a JSON value that may arrive as a decimal string or a
// bare number into a uint64.
func asUint(v any) (uint64, bool) {
	switch x := v.(type) {
	case string:
		n, err := strconv.ParseUint(x, 10, 64)
		return n, err == nil
	case float64:
		if x < 0 {
			return 0, false
		}
		return uint64(x), true
	default:
		return 0, false
	}
}
