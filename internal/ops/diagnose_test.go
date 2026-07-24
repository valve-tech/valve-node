package ops

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

func diagWire() catalog.WireConfig {
	return catalog.WireConfig{
		ChainID:  369,
		ExecID:   "reth",
		BeaconID: "lighthouse-pulse",
		DataDir:  "/var/lib/valve-node/369",
	}
}

// healthyDiagScripts scripts every probe NetworkDiagnostics runs to its
// healthy answer for diagWire (chain 369 = 0x171, lighthouse p2p 9000).
// Individual tests override single keys to break one probe at a time.
func healthyDiagScripts(e *fakeExecutor) *fakeExecutor {
	return e.
		script("systemctl is-active", executor.Result{Stdout: "active\nactive\n", ExitCode: 0}).
		script("eth_chainId", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x171"}`, ExitCode: 0}).
		script("node/version", executor.Result{Stdout: "200", ExitCode: 0}).
		script("ss -ltn", executor.Result{Stdout: "State Local Address:Port\nLISTEN 0.0.0.0:30303\nLISTEN 0.0.0.0:9000\nLISTEN 127.0.0.1:8545\nLISTEN 0.0.0.0:22\n", ExitCode: 0}).
		script("ss -lun", executor.Result{Stdout: "State Local Address:Port\nUNCONN 0.0.0.0:30303\nUNCONN 0.0.0.0:9000\n", ExitCode: 0}).
		script("checkpoint.pulsechain.com", executor.Result{ExitCode: 0}).
		script("net_peerCount", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x10"}`, ExitCode: 0}).
		script("peer_count", executor.Result{Stdout: `{"data":{"connected":"25","disconnected":"3"}}`, ExitCode: 0}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`, ExitCode: 0}).
		script("node/syncing", executor.Result{Stdout: `{"data":{"head_slot":"123456","sync_distance":"0","is_syncing":false}}`, ExitCode: 0}).
		script("journalctl", executor.Result{Stdout: "", ExitCode: 0})
}

func itemByID(t *testing.T, items []CheckItem, id string) CheckItem {
	t.Helper()
	for _, it := range items {
		if it.ID == id {
			return it
		}
	}
	t.Fatalf("no item %q in %+v", id, items)
	return CheckItem{}
}

func TestNetworkDiagnostics_AllHealthy(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor())
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	wantIDs := []string{
		"diag-services", "diag-exec-rpc", "diag-beacon-api",
		"exec-p2p-open", "beacon-p2p-open",
		"diag-outbound", "diag-exec-peers", "diag-beacon-peers",
		"diag-sync", "diag-journal",
	}
	if len(items) != len(wantIDs) {
		t.Fatalf("got %d items, want %d: %+v", len(items), len(wantIDs), items)
	}
	for i, id := range wantIDs {
		if items[i].ID != id {
			t.Errorf("item[%d].ID = %q, want %q", i, items[i].ID, id)
		}
	}
	for _, it := range items {
		if it.Status != "pass" {
			t.Errorf("item %s = %q (%s), want pass", it.ID, it.Status, it.Detail)
		}
	}
}

// TestNetworkDiagnostics_ServicesDownStopsLadder locks in the ladder
// contract: the suite runs check-by-check and STOPS at the first failure —
// with the services down, nothing after diag-services is probed at all.
func TestNetworkDiagnostics_ServicesDownStopsLadder(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor()).
		script("systemctl is-active", executor.Result{Stdout: "inactive\ninactive\n", ExitCode: 3})
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want the ladder to stop after diag-services: %+v", len(items), items)
	}
	svc := items[0]
	if svc.ID != "diag-services" || svc.Status != "fail" {
		t.Fatalf("items[0] = %s/%s, want diag-services/fail", svc.ID, svc.Status)
	}
	if !strings.Contains(svc.Fix, "systemctl start") || !strings.Contains(svc.Fix, "journalctl") {
		t.Errorf("diag-services Fix %q should suggest systemctl start + journalctl", svc.Fix)
	}
	for _, c := range e.callLog() {
		for _, probe := range []string{"eth_chainId", "net_peerCount", "ss -ltn", "journalctl"} {
			if strings.Contains(c, probe) {
				t.Errorf("probe %q ran after the ladder should have stopped: %q", probe, c)
			}
		}
	}
}

func TestNetworkDiagnostics_ZeroExecPeersFails(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor()).
		script("net_peerCount", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x0"}`, ExitCode: 0})
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	it := itemByID(t, items, "diag-exec-peers")
	if it.Status != "fail" {
		t.Errorf("diag-exec-peers = %q, want fail on 0 peers", it.Status)
	}
	if !strings.Contains(it.Detail, "0 peers") {
		t.Errorf("Detail %q should state the peer count", it.Detail)
	}
	if it.Fix == "" {
		t.Error("want a Fix for 0 peers")
	}
	if last := items[len(items)-1]; last.ID != "diag-exec-peers" {
		t.Errorf("ladder should stop at diag-exec-peers, but last item is %s", last.ID)
	}
}

func TestNetworkDiagnostics_LowPeersWarn(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor()).
		script("net_peerCount", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x2"}`, ExitCode: 0}).
		script("peer_count", executor.Result{Stdout: `{"data":{"connected":"4"}}`, ExitCode: 0})
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	if got := itemByID(t, items, "diag-exec-peers").Status; got != "warn" {
		t.Errorf("diag-exec-peers = %q, want warn on 2 peers", got)
	}
	if got := itemByID(t, items, "diag-beacon-peers").Status; got != "warn" {
		t.Errorf("diag-beacon-peers = %q, want warn on 4 peers", got)
	}
}

func TestNetworkDiagnostics_ChainIDMismatchFails(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor()).
		script("eth_chainId", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x1"}`, ExitCode: 0})
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	it := itemByID(t, items, "diag-exec-rpc")
	if it.Status != "fail" {
		t.Errorf("diag-exec-rpc = %q, want fail on chain-id mismatch", it.Status)
	}
	if !strings.Contains(it.Detail, "369") {
		t.Errorf("Detail %q should mention the expected chain id", it.Detail)
	}
	if last := items[len(items)-1]; last.ID != "diag-exec-rpc" {
		t.Errorf("ladder should stop at diag-exec-rpc, but last item is %s", last.ID)
	}
}

func TestNetworkDiagnostics_OutboundFailureModes(t *testing.T) {
	cases := []struct {
		exit int
		want string
	}{
		{6, "DNS"},
		{7, "refused"},
		{28, "timed out"},
	}
	for _, tc := range cases {
		e := healthyDiagScripts(newFakeExecutor()).
			script("checkpoint.pulsechain.com", executor.Result{ExitCode: tc.exit})
		items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
		if err != nil {
			t.Fatalf("NetworkDiagnostics: %v", err)
		}
		it := itemByID(t, items, "diag-outbound")
		if it.Status != "fail" {
			t.Errorf("exit %d: diag-outbound = %q, want fail", tc.exit, it.Status)
		}
		if !strings.Contains(it.Detail, tc.want) {
			t.Errorf("exit %d: Detail %q should contain %q", tc.exit, it.Detail, tc.want)
		}
		if last := items[len(items)-1]; last.ID != "diag-outbound" {
			t.Errorf("exit %d: ladder should stop at diag-outbound, but last item is %s", tc.exit, last.ID)
		}
	}
}

func TestNetworkDiagnostics_JournalSignatureWarns(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor()).
		script("journalctl", executor.Result{
			Stdout:   "INFO imported chain segment\nWARN low peer count: 0 peers\nWARN low peer count: 0 peers\n",
			ExitCode: 0,
		})
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	it := itemByID(t, items, "diag-journal")
	if it.Status != "warn" {
		t.Errorf("diag-journal = %q, want warn for warn-severity signatures", it.Status)
	}
	if !strings.Contains(it.Detail, "low-peer-count") {
		t.Errorf("Detail %q should name the matched signature", it.Detail)
	}
}

func TestNetworkDiagnostics_InboundDialOnlyInSSHMode(t *testing.T) {
	var dialed []string
	dial := func(network, addr string, timeout time.Duration) error {
		dialed = append(dialed, network+" "+addr)
		if strings.HasSuffix(addr, ":9000") {
			return fmt.Errorf("connect: connection timed out")
		}
		return nil
	}

	// Local mode: never dials, no inbound item.
	e := healthyDiagScripts(newFakeExecutor())
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{Dial: dial})
	if err != nil {
		t.Fatalf("NetworkDiagnostics (local): %v", err)
	}
	if len(dialed) != 0 {
		t.Fatalf("local mode must not dial, dialed %v", dialed)
	}
	for _, it := range items {
		if it.ID == "diag-p2p-inbound" {
			t.Fatal("local mode must not include diag-p2p-inbound")
		}
	}

	// SSH mode: dials both p2p TCP ports from the app host; one failing
	// dial makes the item warn and names the failing port.
	e = healthyDiagScripts(newFakeExecutor())
	items, err = NetworkDiagnostics(context.Background(), e, diagWire(),
		DiagnoseOpts{SSHMode: true, SSHHost: "198.51.100.7", Dial: dial})
	if err != nil {
		t.Fatalf("NetworkDiagnostics (ssh): %v", err)
	}
	wantDials := []string{"tcp 198.51.100.7:30303", "tcp 198.51.100.7:9000"}
	if len(dialed) != len(wantDials) {
		t.Fatalf("dialed %v, want %v", dialed, wantDials)
	}
	for i, want := range wantDials {
		if dialed[i] != want {
			t.Errorf("dial[%d] = %q, want %q", i, dialed[i], want)
		}
	}
	it := itemByID(t, items, "diag-p2p-inbound")
	if it.Status != "warn" {
		t.Errorf("diag-p2p-inbound = %q, want warn when a dial fails", it.Status)
	}
	if !strings.Contains(it.Detail, "9000") {
		t.Errorf("Detail %q should name the unreachable port", it.Detail)
	}
}

func TestNetworkDiagnostics_InboundAllReachablePasses(t *testing.T) {
	dial := func(network, addr string, timeout time.Duration) error { return nil }
	e := healthyDiagScripts(newFakeExecutor())
	items, err := NetworkDiagnostics(context.Background(), e, diagWire(),
		DiagnoseOpts{SSHMode: true, SSHHost: "198.51.100.7", Dial: dial})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	if got := itemByID(t, items, "diag-p2p-inbound").Status; got != "pass" {
		t.Errorf("diag-p2p-inbound = %q, want pass", got)
	}
}

// TestNetworkDiagnostics_StrictlyReadOnly locks in the same contract as
// FirewallChecklist: no command this suite runs may mutate the target.
func TestNetworkDiagnostics_StrictlyReadOnly(t *testing.T) {
	e := healthyDiagScripts(newFakeExecutor())
	if _, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{}); err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	for _, c := range e.callLog() {
		for _, banned := range []string{"ufw allow", "ufw enable", "systemctl start", "systemctl restart", "systemctl stop", "rm ", "chown", "mkdir"} {
			if strings.Contains(c, banned) {
				t.Errorf("mutating command run by diagnostics: %q", c)
			}
		}
	}
}

func TestNetworkDiagnostics_UnknownChainErrors(t *testing.T) {
	w := diagWire()
	w.ChainID = 999999
	if _, err := NetworkDiagnostics(context.Background(), newFakeExecutor(), w, DiagnoseOpts{}); err == nil {
		t.Fatal("want error for unknown chain id, got nil")
	}
}

// TestNetworkDiagnostics_ProbesUseRPCBind locks in Task 2 of the bind-to-host
// work: when the RPC is bound to a routable address (a Tailscale IP), the
// on-box probes must target THAT address, not a hardcoded 127.0.0.1 the
// client no longer listens on.
func TestNetworkDiagnostics_ProbesUseRPCBind(t *testing.T) {
	w := diagWire()
	w.RPCBindAddr = "100.101.102.103"
	e := healthyDiagScripts(newFakeExecutor())
	if _, err := NetworkDiagnostics(context.Background(), e, w, DiagnoseOpts{}); err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	var sawExec, sawBeacon bool
	for _, c := range e.callLog() {
		if strings.Contains(c, "http://100.101.102.103:8545") {
			sawExec = true
		}
		if strings.Contains(c, "http://100.101.102.103:5052") {
			sawBeacon = true
		}
		if strings.Contains(c, "http://127.0.0.1:8545") || strings.Contains(c, "http://127.0.0.1:5052") {
			t.Errorf("probe still hit loopback despite a routable RPC bind: %q", c)
		}
	}
	if !sawExec || !sawBeacon {
		t.Fatalf("expected exec+beacon probes to hit the bind address; sawExec=%v sawBeacon=%v", sawExec, sawBeacon)
	}
}
