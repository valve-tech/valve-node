package setup

import (
	"context"
	"strings"
	"testing"
	"time"

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

// ---- preflight ----

func TestPreflight_FailsOnNonLinux(t *testing.T) {
	e := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Darwin\n", ExitCode: 0})
	step := preflightStep()
	err := step.Verify(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("want error on non-Linux uname, got nil")
	}
	if !strings.Contains(err.Error(), "Linux") {
		t.Fatalf("error %q does not mention Linux", err)
	}
}

func TestPreflight_FailsOnInsufficientDisk(t *testing.T) {
	e := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n", ExitCode: 0}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n1000\n", ExitCode: 0}).
		script("ss -ltn", executor.Result{Stdout: "State  Recv-Q Send-Q Local Address:Port\n", ExitCode: 0})
	step := preflightStep()
	err := step.Verify(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("want error on insufficient disk, got nil")
	}
}

func TestPreflight_FailsOnBusyPort(t *testing.T) {
	e := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n", ExitCode: 0}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n9999999999999\n", ExitCode: 0}).
		script("ss -ltn", executor.Result{
			Stdout:   "State  Recv-Q Send-Q Local Address:Port\nLISTEN 0 128 127.0.0.1:8551 0.0.0.0:*\n",
			ExitCode: 0,
		})
	step := preflightStep()
	err := step.Verify(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("want error on busy port 8551, got nil")
	}
	if !strings.Contains(err.Error(), "8551") {
		t.Fatalf("error %q does not mention the busy port", err)
	}
}

// TestPreflight_ProbesNearestExistingAncestorOnFreshDataDir locks in finding
// 1: DataDir does not exist yet on a fresh box (it's created later by wire's
// mkdir), so the disk probe must walk up to the nearest existing ancestor
// rather than running `df` directly against DataDir. We script the OLD
// plain-df form to FAIL and the NEW ancestor-walk form to succeed; preflight
// must pass, proving the new command shape (not the old one) is what ran.
func TestPreflight_ProbesNearestExistingAncestorOnFreshDataDir(t *testing.T) {
	w := testWire()
	w.DataDir = "/var/lib/valve-node/369"
	e := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n", ExitCode: 0}).
		// The OLD code's exact command shape — must NOT be what's called.
		script("df -B1 --output=avail '/var/lib/valve-node/369'", executor.Result{ExitCode: 1, Stderr: "df: cannot access '/var/lib/valve-node/369': No such file or directory"}).
		// The NEW ancestor-walk command shape.
		script("while [ ! -d", executor.Result{Stdout: "9999999999999\n", ExitCode: 0}).
		script("ss -ltn", executor.Result{Stdout: "State  Recv-Q Send-Q Local Address:Port\n", ExitCode: 0})
	step := preflightStep()
	if err := step.Verify(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("preflight should pass via the ancestor-walk df probe on a fresh box, got %v", err)
	}
}

// TestPreflight_DFAncestorWalkNonZeroExitProducesClearError locks in the
// second half of finding 1: the df command's ExitCode must be checked, and
// a failure must produce a clear error mentioning DataDir (not a bare
// "could not parse df output").
func TestPreflight_DFAncestorWalkNonZeroExitProducesClearError(t *testing.T) {
	w := testWire()
	w.DataDir = "/var/lib/valve-node/369"
	e := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n", ExitCode: 0}).
		script("while [ ! -d", executor.Result{ExitCode: 1, Stderr: "df: boom"})
	step := preflightStep()
	err := step.Verify(context.Background(), e, &State{Wire: w})
	if err == nil {
		t.Fatal("want error when the df ancestor-walk command exits non-zero, got nil")
	}
	if !strings.Contains(err.Error(), w.DataDir) {
		t.Fatalf("error %q does not mention the DataDir path", err)
	}
}

func TestPreflight_PassesWhenAllOK(t *testing.T) {
	e := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n", ExitCode: 0}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n9999999999999\n", ExitCode: 0}).
		script("ss -ltn", executor.Result{Stdout: "State  Recv-Q Send-Q Local Address:Port\n", ExitCode: 0})
	step := preflightStep()
	if err := step.Verify(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if step.Run != nil {
		t.Fatal("preflight has nothing to fix — Run should be nil")
	}
}

// ---- install ----

func TestInstall_SourceBuild_RunsGitCloneBuildCmd(t *testing.T) {
	client, ok := catalog.ClientByID("reth")
	if !ok {
		t.Fatal("catalog missing reth")
	}
	e := newFakeExecutor().
		script("test -x '/usr/local/bin/reth'", executor.Result{ExitCode: 1}). // not installed yet
		script(client.BuildCmd, executor.Result{ExitCode: 0, Stdout: "built\n"})

	step := installStep("install-exec", "Install execution client", client)
	events := make(chan Event, 100)
	st := &State{Wire: testWire(), Events: events}

	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, c := range e.callLog() {
		if c == client.BuildCmd {
			found = true
		}
	}
	if !found {
		t.Fatalf("BuildCmd was not run verbatim; calls = %v", e.callLog())
	}
}

func TestInstall_ReleaseURL_DownloadsInsteadOfBuilding(t *testing.T) {
	client := catalog.Client{
		ID:   "reth",
		Kind: "exec",
		Repo: "https://example.invalid/reth",
		ReleaseURL: func(goos, goarch, version string) string {
			return "https://example.invalid/reth-" + goos + "-" + goarch
		},
		PinVersion: "v1.2.3",
		BuildCmd:   "git clone https://example.invalid/reth && cd reth && cargo build --release",
	}
	e := newFakeExecutor().
		script("curl -fL", executor.Result{ExitCode: 0})

	step := installStep("install-exec", "Install execution client", client)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}

	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var ranBuild, ranCurl bool
	for _, c := range e.callLog() {
		if strings.Contains(c, "cargo build") {
			ranBuild = true
		}
		if strings.Contains(c, "curl -fL") {
			ranCurl = true
		}
	}
	if ranBuild {
		t.Fatal("BuildCmd should not run when ReleaseURL is set")
	}
	if !ranCurl {
		t.Fatalf("curl download did not run; calls = %v", e.callLog())
	}
}

func TestInstall_MarkerSkipsAlreadyInstalled(t *testing.T) {
	client, ok := catalog.ClientByID("reth")
	if !ok {
		t.Fatal("catalog missing reth")
	}
	e := newFakeExecutor().
		script("test -x '/usr/local/bin/reth'", executor.Result{ExitCode: 0, Stdout: "reth 1.0.0\n"})

	step := installStep("install-exec", "Install execution client", client)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}

	if err := step.Verify(context.Background(), e, st); err != nil {
		t.Fatalf("Verify: want nil (already installed), got %v", err)
	}

	runCalled := false
	events := make(chan Event, 100)
	st2 := &State{Wire: testWire(), Events: events}
	steps := []Step{{
		ID:     step.ID,
		Verify: step.Verify,
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			runCalled = true
			return step.Run(ctx, e, st)
		},
	}}
	if err := RunAll(context.Background(), e, steps, st2); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if runCalled {
		t.Fatal("Run executed even though the install marker already succeeded")
	}
}

// TestInstall_ProbesBinaryAtClientIDPath locks in the task-4b fix: every
// client's BuildCmd installs to /usr/local/bin/<client-id> (go-pulse's
// recipe builds `go build -o /usr/local/bin/go-pulse ./cmd/geth`, not a
// binary named `geth`), so the install marker must probe that same path —
// not an aliased upstream binary name.
func TestInstall_ProbesBinaryAtClientIDPath(t *testing.T) {
	client, ok := catalog.ClientByID("go-pulse")
	if !ok {
		t.Fatal("catalog missing go-pulse")
	}
	e := newFakeExecutor().
		script("test -x '/usr/local/bin/go-pulse'", executor.Result{ExitCode: 0, Stdout: "go-pulse 1.0\n"})
	step := installStep("install-exec", "Install execution client", client)
	err := step.Verify(context.Background(), e, &State{Wire: testWire(), Events: make(chan Event, 10)})
	if err != nil {
		t.Fatalf("go-pulse install marker should probe /usr/local/bin/go-pulse: %v", err)
	}
}

// ---- wire ----

func TestWire_WritesJwtOnlyIfAbsent(t *testing.T) {
	w := testWire()
	e := newFakeExecutor().
		script("test -f '/mnt/reth/jwt.hex'", executor.Result{ExitCode: 1}). // absent
		script("openssl rand -hex 32", executor.Result{ExitCode: 0}).
		script("systemctl daemon-reload", executor.Result{ExitCode: 0})

	step := wireStep()
	st := &State{Wire: w, Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wroteJwt := false
	for _, c := range e.callLog() {
		if strings.Contains(c, "openssl rand -hex 32") {
			wroteJwt = true
		}
	}
	if !wroteJwt {
		t.Fatalf("jwt was not written; calls = %v", e.callLog())
	}
	if _, ok := e.files["/etc/systemd/system/valve-node-exec.service"]; !ok {
		t.Fatal("exec unit was not written")
	}
	if _, ok := e.files["/etc/systemd/system/valve-node-beacon.service"]; !ok {
		t.Fatal("beacon unit was not written")
	}
}

// TestWire_JWTCreationSetsUmaskBeforeWriting locks in finding 3: the JWT
// secret file must never exist, even momentarily, with the process umask's
// default (widely-readable) permissions — `umask 077` must be set in the
// same command before openssl writes the file, not applied after the fact
// via a following chmod.
func TestWire_JWTCreationSetsUmaskBeforeWriting(t *testing.T) {
	w := testWire()
	e := newFakeExecutor().
		script("test -f '/mnt/reth/jwt.hex'", executor.Result{ExitCode: 1}). // absent
		script("umask 077 && mkdir -p", executor.Result{ExitCode: 0}).
		script("systemctl daemon-reload", executor.Result{ExitCode: 0})

	step := wireStep()
	st := &State{Wire: w, Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "umask 077 && ") && strings.Contains(c, "openssl rand -hex 32") {
			found = true
		}
	}
	if !found {
		t.Fatalf("jwt creation command did not set umask 077 before writing the secret; calls = %v", e.callLog())
	}
}

func TestWire_SkipsJwtWriteWhenAlreadyPresent(t *testing.T) {
	w := testWire()
	e := newFakeExecutor().
		script("test -f '/mnt/reth/jwt.hex'", executor.Result{ExitCode: 0}). // present
		script("systemctl daemon-reload", executor.Result{ExitCode: 0})

	step := wireStep()
	st := &State{Wire: w, Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, c := range e.callLog() {
		if strings.Contains(c, "openssl rand -hex 32") {
			t.Fatalf("jwt was regenerated even though it already existed; calls = %v", e.callLog())
		}
	}
}

func TestWire_EnablesBothUnits(t *testing.T) {
	w := testWire()
	e := newFakeExecutor().
		script("test -f '/mnt/reth/jwt.hex'", executor.Result{ExitCode: 1}).
		script("openssl rand -hex 32", executor.Result{ExitCode: 0}).
		script("systemctl daemon-reload", executor.Result{ExitCode: 0})

	step := wireStep()
	st := &State{Wire: w, Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	enabled := false
	for _, c := range e.callLog() {
		if strings.Contains(c, "systemctl") && strings.Contains(c, "enable") && strings.Contains(c, "--now") {
			enabled = true
		}
	}
	if !enabled {
		t.Fatalf("systemctl enable --now was not run for both units; calls = %v", e.callLog())
	}
}

// ---- handshake ----

func TestHandshake_PassesWhenAllChecksGood(t *testing.T) {
	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl -u valve-node-beacon.service", executor.Result{Stdout: "beacon: synced ok\n"})

	step := handshakeStep()
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	if err := step.Verify(context.Background(), e, st); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestHandshake_FailureIncludesOffendingLogLines(t *testing.T) {
	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl -u valve-node-beacon.service", executor.Result{
			Stdout: "beacon: ok line\nERR engine api: 401 Unauthorized: bad jwt\nbeacon: another ok line\n",
		})

	step := handshakeStep()
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	err := step.Verify(context.Background(), e, st)
	if err == nil {
		t.Fatal("want handshake error on jwt/401 journal lines, got nil")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized: bad jwt") {
		t.Fatalf("error does not embed the offending journalctl line: %v", err)
	}
}

// TestHandshake_BeaconProbeNonZeroExitTreatedAsNotUpYet locks in finding 5:
// the beacon syncing curl's ExitCode must be checked. A transport-level
// curl failure (e.g. connection refused) must be treated as "not up yet"
// even if Stdout happens to read "200" (curl's -w format string is written
// regardless of transport success), and the error must carry exit/stderr
// context.
func TestHandshake_BeaconProbeNonZeroExitTreatedAsNotUpYet(t *testing.T) {
	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200", ExitCode: 7, Stderr: "curl: (7) Failed to connect"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl -u valve-node-beacon.service", executor.Result{Stdout: "ok\n"})

	err := handshakeCheck(context.Background(), e, testWire(), nil)
	if err == nil {
		t.Fatal("want error when the beacon curl exits non-zero, even if stdout happens to read 200")
	}
	if !strings.Contains(err.Error(), "7") || !strings.Contains(err.Error(), "Failed to connect") {
		t.Fatalf("error should embed exit code and stderr context: %v", err)
	}
}

// TestHandshake_ExecProbeNonZeroExitTreatedAsNotUpYet mirrors the beacon
// case for the execution client's eth_syncing curl probe.
func TestHandshake_ExecProbeNonZeroExitTreatedAsNotUpYet(t *testing.T) {
	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`, ExitCode: 7, Stderr: "curl: (7) Failed to connect"}).
		script("journalctl -u valve-node-beacon.service", executor.Result{Stdout: "ok\n"})

	err := handshakeCheck(context.Background(), e, testWire(), nil)
	if err == nil {
		t.Fatal("want error when the exec curl exits non-zero, even if stdout looks like a valid RPC answer")
	}
	if !strings.Contains(err.Error(), "7") || !strings.Contains(err.Error(), "Failed to connect") {
		t.Fatalf("error should embed exit code and stderr context: %v", err)
	}
}

// TestHandshake_JournalctlNonZeroExitDoesNotFailHandshake locks in the
// second half of finding 5: journalctl legitimately exits non-zero (e.g. a
// grep-filtered pipeline with no matching lines) — that must read as "no
// lines", not a handshake failure.
func TestHandshake_JournalctlNonZeroExitDoesNotFailHandshake(t *testing.T) {
	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl -u valve-node-beacon.service", executor.Result{Stdout: "", ExitCode: 1})

	if err := handshakeCheck(context.Background(), e, testWire(), nil); err != nil {
		t.Fatalf("journalctl exiting non-zero with no matched lines should not fail handshake: %v", err)
	}
}

func TestHandshake_RunPollsUntilSuccessThenReturns(t *testing.T) {
	old := handshakePollInterval
	handshakePollInterval = time.Millisecond
	defer func() { handshakePollInterval = old }()

	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl -u valve-node-beacon.service", executor.Result{Stdout: "ok\n"})

	step := handshakeStep()
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	start := time.Now()
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("Run took too long for an immediately-successful handshake: %s", time.Since(start))
	}
}

func TestHandshake_RunTimesOutWithOffendingLines(t *testing.T) {
	oldTimeout, oldInterval := handshakeTimeout, handshakePollInterval
	handshakeTimeout = 5 * time.Millisecond
	handshakePollInterval = time.Millisecond
	defer func() { handshakeTimeout, handshakePollInterval = oldTimeout, oldInterval }()

	e := newFakeExecutor().
		script("eth/v1/node/syncing", executor.Result{Stdout: "200"}).
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("journalctl -u valve-node-beacon.service", executor.Result{
			Stdout: "ERR 401 unauthorized: jwt mismatch\n",
		})

	step := handshakeStep()
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	err := step.Run(context.Background(), e, st)
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "401 unauthorized: jwt mismatch") {
		t.Fatalf("timeout error does not embed the offending journalctl line: %v", err)
	}
}

// ---- Plan ----

func TestPlan_ReturnsOrderedSteps(t *testing.T) {
	steps, err := Plan(testWire())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := []string{"preflight", "toolchain", "install-exec", "install-beacon", "wire", "start", "handshake"}
	if len(steps) != len(want) {
		t.Fatalf("got %d steps, want %d", len(steps), len(want))
	}
	for i, id := range want {
		if steps[i].ID != id {
			t.Fatalf("step %d ID = %q, want %q", i, steps[i].ID, id)
		}
	}
}

func TestPlan_InvalidComboErrors(t *testing.T) {
	_, err := Plan(catalog.WireConfig{
		ChainID:  1,
		ExecID:   "go-pulse",
		BeaconID: "lighthouse",
		DataDir:  "/mnt/reth",
	})
	if err == nil {
		t.Fatal("want error for go-pulse on chain 1, got nil")
	}
}
