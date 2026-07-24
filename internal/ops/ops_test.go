package ops

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

func testWire() catalog.WireConfig {
	return catalog.WireConfig{
		ChainID:  369,
		ExecID:   "reth",
		BeaconID: "lighthouse-pulse",
		DataDir:  "/mnt/reth",
		Archive:  false,
	}
}

// ---- ServiceAction ----

func TestServiceAction_StartRunsSystemctlAndReadsBack(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl start valve-node-exec.service", executor.Result{ExitCode: 0}).
		script("systemctl is-active valve-node-exec.service", executor.Result{Stdout: "active\n", ExitCode: 0})

	active, err := ServiceAction(context.Background(), e, "exec", "start")
	if err != nil {
		t.Fatalf("ServiceAction: %v", err)
	}
	if !active {
		t.Fatal("want active=true after start + is-active reads back \"active\"")
	}
}

func TestServiceAction_StopReadsBackInactive(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl stop valve-node-beacon.service", executor.Result{ExitCode: 0}).
		script("systemctl is-active valve-node-beacon.service", executor.Result{Stdout: "inactive\n", ExitCode: 3})

	active, err := ServiceAction(context.Background(), e, "beacon", "stop")
	if err != nil {
		t.Fatalf("ServiceAction: %v", err)
	}
	if active {
		t.Fatal("want active=false after stop + is-active reads back \"inactive\" (non-zero exit is not an error)")
	}
}

func TestServiceAction_RestartRunsCorrectUnit(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl restart valve-node-exec.service", executor.Result{ExitCode: 0}).
		script("systemctl is-active valve-node-exec.service", executor.Result{Stdout: "active\n"})

	if _, err := ServiceAction(context.Background(), e, "exec", "restart"); err != nil {
		t.Fatalf("ServiceAction: %v", err)
	}
	found := false
	for _, c := range e.callLog() {
		if c == "systemctl restart valve-node-exec.service" {
			found = true
		}
	}
	if !found {
		t.Fatalf("restart did not run against the exec unit; calls = %v", e.callLog())
	}
}

func TestServiceAction_InvalidServiceErrors(t *testing.T) {
	e := newFakeExecutor()
	if _, err := ServiceAction(context.Background(), e, "bogus", "start"); err == nil {
		t.Fatal("want error for invalid svc, got nil")
	}
	if len(e.callLog()) != 0 {
		t.Fatalf("invalid svc must not run anything; calls = %v", e.callLog())
	}
}

func TestServiceAction_InvalidActionErrors(t *testing.T) {
	e := newFakeExecutor()
	if _, err := ServiceAction(context.Background(), e, "exec", "bogus"); err == nil {
		t.Fatal("want error for invalid action, got nil")
	}
	if len(e.callLog()) != 0 {
		t.Fatalf("invalid action must not run anything; calls = %v", e.callLog())
	}
}

func TestServiceAction_NonZeroExitOnCommandIsError(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl start valve-node-exec.service", executor.Result{ExitCode: 1, Stderr: "Unit not found."})
	if _, err := ServiceAction(context.Background(), e, "exec", "start"); err == nil {
		t.Fatal("want error when systemctl start exits non-zero")
	}
}

// ---- ClearService ----

func TestClearService_StopRmStartOrder(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl stop valve-node-exec.service", executor.Result{ExitCode: 0}).
		script("rm -rf", executor.Result{ExitCode: 0}).
		script("systemctl start valve-node-exec.service", executor.Result{ExitCode: 0})

	if err := ClearService(context.Background(), e, testWire(), "exec"); err != nil {
		t.Fatalf("ClearService: %v", err)
	}

	calls := e.callLog()
	var stopIdx, rmIdx, startIdx = -1, -1, -1
	for i, c := range calls {
		switch {
		case strings.Contains(c, "systemctl stop"):
			stopIdx = i
		case strings.HasPrefix(c, "rm -rf"):
			rmIdx = i
		case strings.Contains(c, "systemctl start"):
			startIdx = i
		}
	}
	if stopIdx == -1 || rmIdx == -1 || startIdx == -1 {
		t.Fatalf("missing expected calls; calls = %v", calls)
	}
	if !(stopIdx < rmIdx && rmIdx < startIdx) {
		t.Fatalf("want stop < rm < start order, got stop=%d rm=%d start=%d; calls = %v", stopIdx, rmIdx, startIdx, calls)
	}
}

func TestClearService_DeletesOnlyClientOwnedSubdirs(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl stop", executor.Result{ExitCode: 0}).
		script("rm -rf", executor.Result{ExitCode: 0}).
		script("systemctl start", executor.Result{ExitCode: 0})

	if err := ClearService(context.Background(), e, testWire(), "exec"); err != nil {
		t.Fatalf("ClearService: %v", err)
	}

	var rmCmd string
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "rm -rf") {
			rmCmd = c
		}
	}
	// reth's DataSubdirs are db + static_files.
	if !strings.Contains(rmCmd, shQuote("/mnt/reth/db")) {
		t.Errorf("rm command %q does not delete /mnt/reth/db", rmCmd)
	}
	if !strings.Contains(rmCmd, shQuote("/mnt/reth/static_files")) {
		t.Errorf("rm command %q does not delete /mnt/reth/static_files", rmCmd)
	}
	if strings.Contains(rmCmd, "jwt.hex") {
		t.Errorf("rm command %q must never touch jwt.hex", rmCmd)
	}
	// beacon's data (lighthouse-pulse: DataDir/beacon) must survive an
	// exec clear.
	if strings.Contains(rmCmd, shQuote("/mnt/reth/beacon")) {
		t.Errorf("rm command %q deleted the sibling beacon client's data", rmCmd)
	}
}

func TestClearService_BeaconClearDeletesOnlyBeaconSubdir(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl stop", executor.Result{ExitCode: 0}).
		script("rm -rf", executor.Result{ExitCode: 0}).
		script("systemctl start", executor.Result{ExitCode: 0})

	if err := ClearService(context.Background(), e, testWire(), "beacon"); err != nil {
		t.Fatalf("ClearService: %v", err)
	}
	var rmCmd string
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "rm -rf") {
			rmCmd = c
		}
	}
	if !strings.Contains(rmCmd, shQuote("/mnt/reth/beacon")) {
		t.Errorf("rm command %q does not delete /mnt/reth/beacon", rmCmd)
	}
	if strings.Contains(rmCmd, shQuote("/mnt/reth/db")) {
		t.Errorf("rm command %q deleted the sibling exec client's data", rmCmd)
	}
}

func TestClearService_RmFailureAbortsAndSkipsStart(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl stop", executor.Result{ExitCode: 0}).
		script("rm -rf", executor.Result{ExitCode: 1, Stderr: "rm: cannot remove: Permission denied"})

	err := ClearService(context.Background(), e, testWire(), "exec")
	if err == nil {
		t.Fatal("want error when rm -rf fails, got nil")
	}
	if !strings.Contains(err.Error(), "unit left stopped") {
		t.Errorf("error %q should say the unit was left stopped", err)
	}
	for _, c := range e.callLog() {
		if strings.Contains(c, "systemctl start") {
			t.Fatalf("start must not run after a failed rm; calls = %v", e.callLog())
		}
	}
}

func TestClearService_StopFailureSkipsDeleteAndStart(t *testing.T) {
	e := newFakeExecutor().
		script("systemctl stop", executor.Result{ExitCode: 1, Stderr: "boom"})

	err := ClearService(context.Background(), e, testWire(), "exec")
	if err == nil {
		t.Fatal("want error when systemctl stop fails, got nil")
	}
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "rm -rf") || strings.Contains(c, "systemctl start") {
			t.Fatalf("delete/start must not run after a failed stop; calls = %v", e.callLog())
		}
	}
}

func TestClearService_InvalidServiceErrors(t *testing.T) {
	e := newFakeExecutor()
	if err := ClearService(context.Background(), e, testWire(), "bogus"); err == nil {
		t.Fatal("want error for invalid svc")
	}
	if len(e.callLog()) != 0 {
		t.Fatalf("invalid svc must not run anything; calls = %v", e.callLog())
	}
}

func TestClearService_UnsafeDataDirRefusesWithZeroExecutorCalls(t *testing.T) {
	// An unsafe DataDir must be caught by clearPaths before ClearService
	// ever touches the Executor — not even the systemctl stop should run.
	w := testWire()
	w.DataDir = ""
	e := newFakeExecutor()

	if err := ClearService(context.Background(), e, w, "exec"); err == nil {
		t.Fatal("want error for an unsafe DataDir, got nil")
	}
	if calls := e.callLog(); len(calls) != 0 {
		t.Fatalf("unsafe DataDir must not run ANY executor calls (not even stop); calls = %v", calls)
	}
}

// ---- clearPaths safety ----

func TestClearPaths_RefusesEmptyDataDir(t *testing.T) {
	if _, err := clearPaths("", []string{"geth"}); err == nil {
		t.Fatal("want error for empty DataDir")
	}
}

func TestClearPaths_RefusesRootDataDir(t *testing.T) {
	if _, err := clearPaths("/", []string{"geth"}); err == nil {
		t.Fatal("want error when DataDir is \"/\"")
	}
}

func TestClearPaths_RefusesWhenSubdirResolvesToDataDirItself(t *testing.T) {
	// An empty subdir (or ".") makes the computed path equal DataDir.
	if _, err := clearPaths("/mnt/reth", []string{""}); err == nil {
		t.Fatal("want error when a subdir is empty (computed path == DataDir)")
	}
	if _, err := clearPaths("/mnt/reth", []string{"."}); err == nil {
		t.Fatal("want error when a subdir is \".\" (computed path == DataDir)")
	}
}

func TestClearPaths_RefusesWhenComputedPathIsRoot(t *testing.T) {
	// DataDir "/var" with subdir ".." computes to "/".
	if _, err := clearPaths("/var", []string{".."}); err == nil {
		t.Fatal("want error when a computed path resolves to \"/\"")
	}
}

func TestClearPaths_RefusesNoDataSubdirs(t *testing.T) {
	if _, err := clearPaths("/mnt/reth", nil); err == nil {
		t.Fatal("want error when the client has no known DataSubdirs")
	}
}

func TestClearPaths_RefusesSubdirEscapingDataDir(t *testing.T) {
	// DataDir "/mnt/reth" with subdir "../rethbackup" cleans to
	// "/mnt/rethbackup" — a sibling directory, not anything under
	// DataDir. The old exact-match denylist (only checking against
	// DataDir itself, "/", or "") let this slip through; clearPaths must
	// refuse anything that isn't structurally INSIDE DataDir.
	if _, err := clearPaths("/mnt/reth", []string{"../rethbackup"}); err == nil {
		t.Fatal("want error when a computed delete path escapes DataDir to a sibling directory")
	}
}

func TestClearPaths_HappyPathIsDataDirSlashSubdir(t *testing.T) {
	paths, err := clearPaths("/mnt/reth", []string{"db", "static_files"})
	if err != nil {
		t.Fatalf("clearPaths: %v", err)
	}
	want := []string{"/mnt/reth/db", "/mnt/reth/static_files"}
	if len(paths) != len(want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

// ---- DiskUsage ----

func TestDiskUsage_ParsesDuAndDf(t *testing.T) {
	e := newFakeExecutor().
		script("du -sb '/mnt/reth/db' '/mnt/reth/static_files'", executor.Result{
			Stdout: "1000000\t/mnt/reth/db\n2000000\t/mnt/reth/static_files\n", ExitCode: 0,
		}).
		script("du -sb '/mnt/reth/beacon'", executor.Result{Stdout: "500000\t/mnt/reth/beacon\n", ExitCode: 0}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n9999999999\n", ExitCode: 0})

	du, err := DiskUsage(context.Background(), e, testWire())
	if err != nil {
		t.Fatalf("DiskUsage: %v", err)
	}
	if du.ExecBytes != 3000000 {
		t.Errorf("ExecBytes = %d, want 3000000", du.ExecBytes)
	}
	if du.BeaconBytes != 500000 {
		t.Errorf("BeaconBytes = %d, want 500000", du.BeaconBytes)
	}
	if du.DiskFreeBytes != 9999999999 {
		t.Errorf("DiskFreeBytes = %d, want 9999999999", du.DiskFreeBytes)
	}
	wantExpected, _ := catalog.ExpectedBytes(369, false)
	if du.ExpectedExecBytes != wantExpected {
		t.Errorf("ExpectedExecBytes = %d, want %d", du.ExpectedExecBytes, wantExpected)
	}
	if du.ExpectedBeaconBytes == 0 {
		t.Error("ExpectedBeaconBytes should be a non-zero estimate")
	}
	if du.SyncLabel == "" || du.GenesisSyncLabel == "" {
		t.Error("SyncLabel/GenesisSyncLabel should be populated from the catalog")
	}
}

func TestDiskUsage_MissingSubdirTreatedAsZeroNotError(t *testing.T) {
	// du exits non-zero when a subdir doesn't exist yet, but still prints
	// whatever it could measure — the exit code must be ignored.
	e := newFakeExecutor().
		script("du -sb", executor.Result{
			Stdout:   "1000000\t/mnt/reth/db\n",
			Stderr:   "du: cannot access '/mnt/reth/static_files': No such file or directory\n",
			ExitCode: 1,
		}).
		script("df -B1 --output=avail", executor.Result{Stdout: "9999999999\n", ExitCode: 0})

	du, err := DiskUsage(context.Background(), e, testWire())
	if err != nil {
		t.Fatalf("DiskUsage: %v", err)
	}
	if du.ExecBytes != 1000000 {
		t.Errorf("ExecBytes = %d, want 1000000 (only the existing subdir counted, non-zero exit ignored)", du.ExecBytes)
	}
}

func TestDiskUsage_UnknownChainErrors(t *testing.T) {
	w := testWire()
	w.ChainID = 9999
	e := newFakeExecutor()
	if _, err := DiskUsage(context.Background(), e, w); err == nil {
		t.Fatal("want error for unknown chain id")
	}
}

// ---- Endpoints ----

func TestEndpoints_LocalAccessNoTunnelHint(t *testing.T) {
	e := newFakeExecutor().
		script("eth_chainId", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x171"}`, ExitCode: 0}).
		script("/eth/v1/node/version", executor.Result{Stdout: "200", ExitCode: 0})

	ep, err := Endpoints(context.Background(), e, testWire(), false, "")
	if err != nil {
		t.Fatalf("Endpoints: %v", err)
	}
	if ep.Access != "local" {
		t.Errorf("Access = %q, want \"local\"", ep.Access)
	}
	if ep.TunnelHint != "" {
		t.Errorf("TunnelHint = %q, want empty for a local target", ep.TunnelHint)
	}
	if ep.ExecHTTP != "http://127.0.0.1:8545" {
		t.Errorf("ExecHTTP = %q, want the default exec HTTP URL", ep.ExecHTTP)
	}
	if ep.BeaconHTTP != "http://127.0.0.1:5052" {
		t.Errorf("BeaconHTTP = %q, want the default beacon HTTP URL", ep.BeaconHTTP)
	}
	if !ep.ExecReachable || !ep.BeaconReachable {
		t.Errorf("ExecReachable=%v BeaconReachable=%v, want true/true", ep.ExecReachable, ep.BeaconReachable)
	}
	// 0x171 == 369, testWire's ChainID.
	if !ep.ChainIDMatches {
		t.Error("ChainIDMatches = false, want true (0x171 == 369)")
	}
}

func TestEndpoints_SSHAccessIncludesTunnelHint(t *testing.T) {
	e := newFakeExecutor()
	ep, err := Endpoints(context.Background(), e, testWire(), true, "root@203.0.113.5")
	if err != nil {
		t.Fatalf("Endpoints: %v", err)
	}
	if ep.Access != "ssh" {
		t.Errorf("Access = %q, want \"ssh\"", ep.Access)
	}
	// Exact match: the sshLogin argument is the full user@host login,
	// embedded verbatim — no "root@" is ever prefixed by Endpoints itself
	// (that was the bug: a hardcoded "root@%s" on top of an
	// already-user@host-shaped hint produced "root@root@<host>").
	want := "ssh -L 8545:127.0.0.1:8545 -L 5052:127.0.0.1:5052 root@203.0.113.5"
	if ep.TunnelHint != want {
		t.Errorf("TunnelHint = %q, want %q", ep.TunnelHint, want)
	}
}

func TestEndpoints_ChainIDMismatch(t *testing.T) {
	e := newFakeExecutor().
		script("eth_chainId", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x1"}`, ExitCode: 0})

	ep, err := Endpoints(context.Background(), e, testWire(), false, "")
	if err != nil {
		t.Fatalf("Endpoints: %v", err)
	}
	if !ep.ExecReachable {
		t.Error("ExecReachable should be true (the probe answered)")
	}
	if ep.ChainIDMatches {
		t.Error("ChainIDMatches should be false (0x1 == 1, wire wants 369)")
	}
}

func TestEndpoints_UnreachableLeavesFalseNotError(t *testing.T) {
	e := newFakeExecutor().
		errOn("eth_chainId", fmt.Errorf("ssh transport dropped")).
		script("/eth/v1/node/version", executor.Result{ExitCode: 7, Stderr: "curl: (7) Failed to connect"})

	ep, err := Endpoints(context.Background(), e, testWire(), false, "")
	if err != nil {
		t.Fatalf("Endpoints should not error on unreachable probes, got %v", err)
	}
	if ep.ExecReachable || ep.BeaconReachable {
		t.Errorf("ExecReachable=%v BeaconReachable=%v, want false/false", ep.ExecReachable, ep.BeaconReachable)
	}
}

func TestEndpoints_UsesConfiguredPorts(t *testing.T) {
	w := testWire()
	w.ExecHTTPPort = 9545
	w.BeaconHTTPPort = 6052
	e := newFakeExecutor()
	ep, err := Endpoints(context.Background(), e, w, false, "")
	if err != nil {
		t.Fatalf("Endpoints: %v", err)
	}
	if ep.ExecHTTP != "http://127.0.0.1:9545" {
		t.Errorf("ExecHTTP = %q, want the configured custom port", ep.ExecHTTP)
	}
	if ep.BeaconHTTP != "http://127.0.0.1:6052" {
		t.Errorf("BeaconHTTP = %q, want the configured custom port", ep.BeaconHTTP)
	}
}

// ---- FirewallChecklist ----

func ssLine(addr string, port int) string {
	return fmt.Sprintf("LISTEN 0 128 %s:%d 0.0.0.0:*\n", addr, port)
}

func firewallWire() catalog.WireConfig {
	return testWire() // reth/lighthouse-pulse, ExecP2P defaults to 30303
}

func TestFirewall_ExecP2P_WideBindPasses(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 30303)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "exec-p2p-open")
	if it.Status != "pass" {
		t.Errorf("exec-p2p-open Status = %q, want pass; detail=%q", it.Status, it.Detail)
	}
}

func TestFirewall_ExecP2P_LoopbackOnlyWarns(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() + ssLine("127.0.0.1", 30303)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "exec-p2p-open")
	if it.Status != "warn" {
		t.Errorf("exec-p2p-open Status = %q, want warn", it.Status)
	}
	if !strings.Contains(it.Detail, "peers can't reach you") {
		t.Errorf("exec-p2p-open Detail %q missing the risk sentence", it.Detail)
	}
}

func TestFirewall_ExecP2P_NotListeningIsUnknown(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader()}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "exec-p2p-open")
	if it.Status != "unknown" {
		t.Errorf("exec-p2p-open Status = %q, want unknown", it.Status)
	}
}

func TestFirewall_BeaconP2P_UsesClientFamilyPorts(t *testing.T) {
	// lighthouse-pulse: 9000 tcp+udp.
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 9000)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "beacon-p2p-open")
	if it.Status != "pass" {
		t.Errorf("beacon-p2p-open Status = %q, want pass; detail=%q", it.Status, it.Detail)
	}
}

func TestFirewall_BeaconP2P_PrysmFamilyDifferentTCPUDPPorts(t *testing.T) {
	w := firewallWire()
	w.BeaconID = "prysm-pulse"
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 13000)}).
		script("ss -lun", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 12000)}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, w)
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "beacon-p2p-open")
	if it.Status != "pass" {
		t.Errorf("beacon-p2p-open Status = %q, want pass; detail=%q", it.Status, it.Detail)
	}
}

func TestFirewall_RPCNotPublic_LoopbackPasses(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() +
			ssLine("127.0.0.1", 8545) + ssLine("127.0.0.1", 8551) + ssLine("127.0.0.1", 5052)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "rpc-not-public")
	if it.Status != "pass" {
		t.Errorf("rpc-not-public Status = %q, want pass; detail=%q", it.Status, it.Detail)
	}
}

func TestFirewall_RPCNotPublic_WideBindFails(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() +
			ssLine("0.0.0.0", 8545) + ssLine("127.0.0.1", 8551) + ssLine("127.0.0.1", 5052)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "rpc-not-public")
	if it.Status != "fail" {
		t.Errorf("rpc-not-public Status = %q, want fail; detail=%q", it.Status, it.Detail)
	}
	if !strings.Contains(it.Detail, "8545") {
		t.Errorf("rpc-not-public Detail %q should name the offending port", it.Detail)
	}
}

func TestFirewall_UfwInactive_WarnsWithSSHFirstFixBlock(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() + ssLine("127.0.0.1", 22)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: inactive\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "firewall-active")
	if it.Status != "warn" {
		t.Errorf("firewall-active Status = %q, want warn", it.Status)
	}
	lines := strings.Split(strings.TrimSpace(it.Fix), "\n")
	if len(lines) == 0 || lines[0] != "ufw allow 22/tcp" {
		t.Fatalf("firewall-active Fix must start with \"ufw allow 22/tcp\", got %q", it.Fix)
	}
}

func TestFirewall_SSHListening_Passes(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 22)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "ssh-allowed")
	if it.Status != "pass" {
		t.Errorf("ssh-allowed Status = %q, want pass", it.Status)
	}
}

func TestFirewall_SSHNotListening_Warns(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader()}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "ssh-allowed")
	if it.Status != "warn" {
		t.Errorf("ssh-allowed Status = %q, want warn", it.Status)
	}
}

func TestFirewall_NeverMutates(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader()}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: inactive\n", ExitCode: 0})

	if _, err := FirewallChecklist(context.Background(), e, firewallWire()); err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	for _, c := range e.callLog() {
		for _, banned := range []string{"ufw allow", "ufw enable", "iptables -A"} {
			if strings.Contains(c, banned) {
				t.Fatalf("FirewallChecklist ran a mutating command: %q (matched %q)", c, banned)
			}
		}
	}
}

func TestFirewall_ReturnsAllFiveItems(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader()}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	wantIDs := []string{"exec-p2p-open", "beacon-p2p-open", "rpc-not-public", "firewall-active", "ssh-allowed"}
	if len(items) != len(wantIDs) {
		t.Fatalf("got %d items, want %d: %+v", len(items), len(wantIDs), items)
	}
	for _, id := range wantIDs {
		findItem(t, items, id)
	}
}

func TestFirewall_UnknownBeaconClientErrors(t *testing.T) {
	w := firewallWire()
	w.BeaconID = "bogus-beacon"
	e := newFakeExecutor()
	if _, err := FirewallChecklist(context.Background(), e, w); err == nil {
		t.Fatal("want error for a beacon client with no known p2p ports")
	}
}

func sslHeader() string {
	return "State  Recv-Q Send-Q Local Address:Port  Peer Address:Port\n"
}

func findItem(t *testing.T, items []CheckItem, id string) CheckItem {
	t.Helper()
	for _, it := range items {
		if it.ID == id {
			return it
		}
	}
	t.Fatalf("no checklist item with id %q; items = %+v", id, items)
	return CheckItem{}
}

// ---- bindState ----

func TestBindState_DistinguishesWideLoopbackAndAbsent(t *testing.T) {
	out := sslHeader() + ssLine("0.0.0.0", 8545) + ssLine("127.0.0.1", 8551)
	if got := bindState(out, 8545); got != "wide" {
		t.Errorf("bindState(8545) = %q, want wide", got)
	}
	if got := bindState(out, 8551); got != "loopback" {
		t.Errorf("bindState(8551) = %q, want loopback", got)
	}
	if got := bindState(out, 9999); got != "" {
		t.Errorf("bindState(9999) = %q, want \"\" (not listening)", got)
	}
}

func TestBindState_WideWinsOverLoopbackRegardlessOfOrder(t *testing.T) {
	// A dual-bind service (loopback line listed first, wide line second
	// for the same port) must classify as "wide" — ANY non-loopback
	// listener on the port makes it reachable from outside the box, so
	// first-match-wins on the loopback line was a false "safe" reading.
	out := sslHeader() + ssLine("127.0.0.1", 8545) + ssLine("0.0.0.0", 8545)
	if got := bindState(out, 8545); got != "wide" {
		t.Errorf("bindState(8545) = %q, want wide (dual-bind: loopback line first, wide line second)", got)
	}
}

func TestBindState_DoesNotConfusePortSubstrings(t *testing.T) {
	// A listener on 18551 must not be mistaken for 8551.
	out := sslHeader() + ssLine("0.0.0.0", 18551)
	if got := bindState(out, 8551); got != "" {
		t.Errorf("bindState(8551) = %q, want \"\" (18551 must not match 8551)", got)
	}
}

func TestFirewall_RPCNotPublic_TailscalePassesWithNote(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() +
			ssLine("100.101.102.103", 8545) + ssLine("127.0.0.1", 8551) + ssLine("100.101.102.103", 5052)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "rpc-not-public")
	if it.Status != "pass" {
		t.Errorf("rpc-not-public on a Tailscale bind = %q, want pass; detail=%q", it.Status, it.Detail)
	}
	if !strings.Contains(strings.ToLower(it.Detail), "tailscale") {
		t.Errorf("Detail %q should note the Tailscale/overlay bind", it.Detail)
	}
}

func TestFirewall_RPCNotPublic_LANWarns(t *testing.T) {
	e := newFakeExecutor().
		script("ss -ltn", executor.Result{Stdout: sslHeader() +
			ssLine("192.168.1.10", 8545) + ssLine("127.0.0.1", 8551) + ssLine("127.0.0.1", 5052)}).
		script("ss -lun", executor.Result{Stdout: sslHeader()}).
		script("ufw status", executor.Result{Stdout: "Status: active\n", ExitCode: 0})

	items, err := FirewallChecklist(context.Background(), e, firewallWire())
	if err != nil {
		t.Fatalf("FirewallChecklist: %v", err)
	}
	it := findItem(t, items, "rpc-not-public")
	if it.Status != "warn" {
		t.Errorf("rpc-not-public on a LAN bind = %q, want warn; detail=%q", it.Status, it.Detail)
	}
	if !strings.Contains(it.Detail, "8545") {
		t.Errorf("Detail %q should name the LAN-bound port", it.Detail)
	}
}
