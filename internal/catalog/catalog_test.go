package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node/internal/executor"
)

// allExecIDs / allBeaconIDs enumerate every client id in the catalog by
// kind, independent of which networks they're valid on — used to build
// the invalid-combo matrix below.
func allExecIDs(t *testing.T) []string {
	t.Helper()
	var ids []string
	for _, n := range Networks() {
		ids = append(ids, n.ExecClients...)
	}
	return dedup(ids)
}

func allBeaconIDs(t *testing.T) []string {
	t.Helper()
	var ids []string
	for _, n := range Networks() {
		ids = append(ids, n.BeaconClients...)
	}
	return dedup(ids)
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// authRPCFlag is the exec-unit substring RenderUnits must emit per client
// family, per task-3-brief.md Step 1(a).
func authRPCFlag(execID string) string {
	switch execID {
	case "geth", "go-pulse", "erigon-pulse", "reth":
		return "--authrpc.port"
	default:
		return "--authrpc"
	}
}

func TestRenderUnits_ValidCombos(t *testing.T) {
	for _, net := range Networks() {
		net := net
		for _, execID := range net.ExecClients {
			for _, beaconID := range net.BeaconClients {
				execID, beaconID := execID, beaconID
				t.Run(net.Name+"/"+execID+"+"+beaconID, func(t *testing.T) {
					w := WireConfig{
						ChainID:  net.ChainID,
						ExecID:   execID,
						BeaconID: beaconID,
						DataDir:  "/var/lib/valve-node/dir",
						JWTPath:  "/var/lib/valve-node/dir/jwt.hex",
					}
					execUnit, beaconUnit, err := RenderUnits(w)
					if err != nil {
						t.Fatalf("RenderUnits(%+v) unexpected error: %v", w, err)
					}

					// (c) both start with [Unit] and include Restart=always.
					if !strings.HasPrefix(execUnit, "[Unit]") {
						t.Errorf("exec unit does not start with [Unit]:\n%s", execUnit)
					}
					if !strings.Contains(execUnit, "Restart=always") {
						t.Errorf("exec unit missing Restart=always:\n%s", execUnit)
					}
					if !strings.HasPrefix(beaconUnit, "[Unit]") {
						t.Errorf("beacon unit does not start with [Unit]:\n%s", beaconUnit)
					}
					if !strings.Contains(beaconUnit, "Restart=always") {
						t.Errorf("beacon unit missing Restart=always:\n%s", beaconUnit)
					}
					if !strings.Contains(execUnit, "WantedBy=multi-user.target") {
						t.Errorf("exec unit missing WantedBy=multi-user.target:\n%s", execUnit)
					}
					if !strings.Contains(beaconUnit, "WantedBy=multi-user.target") {
						t.Errorf("beacon unit missing WantedBy=multi-user.target:\n%s", beaconUnit)
					}

					// (a) exec unit: authrpc flag + JWT path.
					if !strings.Contains(execUnit, authRPCFlag(execID)) {
						t.Errorf("exec unit missing %q:\n%s", authRPCFlag(execID), execUnit)
					}
					if !strings.Contains(execUnit, w.JWTPath) {
						t.Errorf("exec unit missing JWT path %q:\n%s", w.JWTPath, execUnit)
					}

					// (b) beacon unit: engine endpoint, JWT path, checkpoint URL.
					if !strings.Contains(beaconUnit, "http://127.0.0.1:8551") {
						t.Errorf("beacon unit missing engine endpoint:\n%s", beaconUnit)
					}
					if !strings.Contains(beaconUnit, w.JWTPath) {
						t.Errorf("beacon unit missing JWT path %q:\n%s", w.JWTPath, beaconUnit)
					}
					if !strings.Contains(beaconUnit, net.CheckpointURL) {
						t.Errorf("beacon unit missing checkpoint URL %q:\n%s", net.CheckpointURL, beaconUnit)
					}
				})
			}
		}
	}
}

func TestRenderUnits_InvalidCombosError(t *testing.T) {
	execIDs := allExecIDs(t)
	beaconIDs := allBeaconIDs(t)

	for _, net := range Networks() {
		net := net
		for _, execID := range execIDs {
			for _, beaconID := range beaconIDs {
				validExec := contains(net.ExecClients, execID)
				validBeacon := contains(net.BeaconClients, beaconID)
				if validExec && validBeacon {
					continue // covered by TestRenderUnits_ValidCombos
				}
				execID, beaconID := execID, beaconID
				t.Run(net.Name+"/"+execID+"+"+beaconID, func(t *testing.T) {
					w := WireConfig{
						ChainID:  net.ChainID,
						ExecID:   execID,
						BeaconID: beaconID,
						DataDir:  "/var/lib/valve-node/dir",
						JWTPath:  "/var/lib/valve-node/dir/jwt.hex",
					}
					_, _, err := RenderUnits(w)
					if err == nil {
						t.Fatalf("RenderUnits(%+v) expected error for invalid combo, got nil", w)
					}
				})
			}
		}
	}
}

func TestRenderUnits_UnknownChainID(t *testing.T) {
	w := WireConfig{ChainID: 999, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/data"}
	if _, _, err := RenderUnits(w); err == nil {
		t.Fatal("expected error for unknown chain id")
	}
}

func TestRenderUnits_UnknownClientID(t *testing.T) {
	w := WireConfig{ChainID: 369, ExecID: "nonexistent", BeaconID: "lighthouse-pulse", DataDir: "/data"}
	if _, _, err := RenderUnits(w); err == nil {
		t.Fatal("expected error for unknown exec client id")
	}
	w2 := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "nonexistent", DataDir: "/data"}
	if _, _, err := RenderUnits(w2); err == nil {
		t.Fatal("expected error for unknown beacon client id")
	}
}

func TestNetworks_LearnURLs(t *testing.T) {
	for _, n := range Networks() {
		if n.LearnURL == "" {
			t.Errorf("network %d (%s) has empty LearnURL", n.ChainID, n.Name)
		}
		if !strings.HasPrefix(n.LearnURL, "https://learn.valve.city/rpc") {
			t.Errorf("network %d (%s) LearnURL %q not under https://learn.valve.city/rpc", n.ChainID, n.Name, n.LearnURL)
		}
	}
}

func TestClients_ResolveAndHaveLearnURLs(t *testing.T) {
	ids := []string{"reth", "go-pulse", "erigon-pulse", "geth", "lighthouse-pulse", "prysm-pulse", "lighthouse"}
	for _, id := range ids {
		c, ok := ClientByID(id)
		if !ok {
			t.Errorf("ClientByID(%q) did not resolve", id)
			continue
		}
		if c.ID != id {
			t.Errorf("ClientByID(%q).ID = %q, want %q", id, c.ID, id)
		}
		if c.Kind != "exec" && c.Kind != "beacon" {
			t.Errorf("client %q has invalid Kind %q", id, c.Kind)
		}
		if c.Repo == "" {
			t.Errorf("client %q has empty Repo", id)
		}
		if c.BuildCmd == "" {
			t.Errorf("client %q has empty BuildCmd", id)
		}
		wantInstallPath := "/usr/local/bin/" + id
		if !strings.Contains(c.BuildCmd, wantInstallPath) {
			t.Errorf("client %q BuildCmd does not install to %q:\n%s", id, wantInstallPath, c.BuildCmd)
		}
		if c.Toolchain != "go" && c.Toolchain != "rust" {
			t.Errorf("client %q has invalid Toolchain %q, want \"go\" or \"rust\"", id, c.Toolchain)
		}
		if c.PinVersion == "" {
			t.Errorf("client %q has empty PinVersion", id)
		}
		if c.ReleaseURL == nil {
			t.Errorf("client %q has nil ReleaseURL", id)
		}
		if c.LearnURL == "" {
			t.Errorf("client %q has empty LearnURL", id)
		}
		if !strings.HasPrefix(c.LearnURL, "https://learn.valve.city/rpc") {
			t.Errorf("client %q LearnURL %q not under https://learn.valve.city/rpc", id, c.LearnURL)
		}
	}
}

func TestNetworkByChainID(t *testing.T) {
	for _, id := range []int{1, 369, 943} {
		n, ok := NetworkByChainID(id)
		if !ok {
			t.Errorf("NetworkByChainID(%d) not found", id)
			continue
		}
		if n.ChainID != id {
			t.Errorf("NetworkByChainID(%d).ChainID = %d", id, n.ChainID)
		}
	}
	if _, ok := NetworkByChainID(1234); ok {
		t.Error("NetworkByChainID(1234) unexpectedly found")
	}
}

func TestRenderUnits_ArchiveFlagAffectsExecUnit(t *testing.T) {
	base := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}

	full := base
	full.Archive = false
	fullExec, _, err := RenderUnits(full)
	if err != nil {
		t.Fatalf("RenderUnits(full) error: %v", err)
	}

	archive := base
	archive.Archive = true
	archiveExec, _, err := RenderUnits(archive)
	if err != nil {
		t.Fatalf("RenderUnits(archive) error: %v", err)
	}

	if fullExec == archiveExec {
		t.Error("Archive=false and Archive=true produced identical exec units")
	}
}

func TestRenderUnits_ErigonPulseArchiveFlags(t *testing.T) {
	base := WireConfig{ChainID: 369, ExecID: "erigon-pulse", BeaconID: "lighthouse-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}

	// Test Archive=true: no --gcmode and no --prune (erigon defaults to archive)
	archive := base
	archive.Archive = true
	archiveExec, _, err := RenderUnits(archive)
	if err != nil {
		t.Fatalf("RenderUnits(Archive=true) error: %v", err)
	}
	if strings.Contains(archiveExec, "--gcmode") {
		t.Errorf("erigon-pulse Archive=true should not contain --gcmode:\n%s", archiveExec)
	}
	if strings.Contains(archiveExec, "--prune") {
		t.Errorf("erigon-pulse Archive=true should not contain --prune:\n%s", archiveExec)
	}

	// Test Archive=false: should contain --prune=hrtc and not contain --gcmode
	full := base
	full.Archive = false
	fullExec, _, err := RenderUnits(full)
	if err != nil {
		t.Fatalf("RenderUnits(Archive=false) error: %v", err)
	}
	if !strings.Contains(fullExec, "--prune=hrtc") {
		t.Errorf("erigon-pulse Archive=false should contain --prune=hrtc:\n%s", fullExec)
	}
	if strings.Contains(fullExec, "--gcmode") {
		t.Errorf("erigon-pulse Archive=false should not contain --gcmode:\n%s", fullExec)
	}
}

// TestRustBuildCmd_CargoEnvSourcing_NoSubshell proves that the rust clients'
// BuildCmd sources ~/.cargo/env in a brace group (current-shell grouping),
// not a subshell — a subshell's PATH edit dies with the subshell, so `cargo
// build` fails with "cargo: not found" on a fresh box where rustup installed
// cargo to ~/.cargo/bin. It also proves the brace-group form still preserves
// short-circuiting: if the preceding clone/cd fails, cargo must never run.
func TestRustBuildCmd_CargoEnvSourcing_NoSubshell(t *testing.T) {
	rustClients := []string{"reth", "lighthouse-pulse", "lighthouse"}
	wantGroup := `{ . "$HOME/.cargo/env" 2>/dev/null || true; } && cargo`

	for _, id := range rustClients {
		id := id
		t.Run(id, func(t *testing.T) {
			c, ok := ClientByID(id)
			if !ok {
				t.Fatalf("ClientByID(%q) not found", id)
			}
			if !strings.Contains(c.BuildCmd, wantGroup) {
				t.Fatalf("client %q BuildCmd does not contain brace-group cargo-env sourcing %q:\n%s", id, wantGroup, c.BuildCmd)
			}

			// Build a scratch HOME with a fake ~/.cargo/env (mimicking rustup's
			// installer) that puts a stub `cargo` on PATH.
			tmp := t.TempDir()
			fakebin := filepath.Join(tmp, "fakebin")
			if err := os.MkdirAll(fakebin, 0o755); err != nil {
				t.Fatal(err)
			}
			cargoEnv := filepath.Join(tmp, ".cargo", "env")
			if err := os.MkdirAll(filepath.Dir(cargoEnv), 0o755); err != nil {
				t.Fatal(err)
			}
			envScript := fmt.Sprintf("export PATH=\"%s:$PATH\"\n", fakebin)
			if err := os.WriteFile(cargoEnv, []byte(envScript), 0o644); err != nil {
				t.Fatal(err)
			}
			cargoStub := filepath.Join(fakebin, "cargo")
			if err := os.WriteFile(cargoStub, []byte("#!/bin/sh\necho cargo-stub-ok\nexit 0\n"), 0o755); err != nil {
				t.Fatal(err)
			}

			exe := executor.NewLocal()
			ctx := context.Background()

			// Property 1: the recipe's literal brace-group form keeps the PATH
			// edit alive past the sourcing statement, so cargo resolves.
			newForm := fmt.Sprintf(`export HOME='%s'; { . "$HOME/.cargo/env" 2>/dev/null || true; } && cargo --version`, tmp)
			res, err := exe.Run(ctx, newForm, nil)
			if err != nil {
				t.Fatalf("brace-group form run error: %v", err)
			}
			if res.ExitCode != 0 {
				t.Errorf("brace-group form exited %d, want 0; stdout=%q stderr=%q", res.ExitCode, res.Stdout, res.Stderr)
			}

			// Discriminator: under the identical scratch HOME, the OLD
			// parenthesized-subshell form must fail, proving this test actually
			// distinguishes the two forms rather than passing regardless.
			oldForm := fmt.Sprintf(`export HOME='%s'; (. "$HOME/.cargo/env" 2>/dev/null || true) && cargo --version`, tmp)
			res2, err := exe.Run(ctx, oldForm, nil)
			if err != nil {
				t.Fatalf("subshell form run error: %v", err)
			}
			if res2.ExitCode == 0 {
				t.Errorf("subshell form unexpectedly exited 0 under scratch HOME — test does not discriminate old vs new")
			}

			// Property 2: short-circuiting is preserved — if the step before the
			// brace group fails, the brace group and everything after it (here,
			// `echo RAN`) must never execute.
			shortCircuit := fmt.Sprintf(`export HOME='%s'; false && { . "$HOME/.cargo/env" 2>/dev/null || true; } && echo RAN`, tmp)
			res3, err := exe.Run(ctx, shortCircuit, nil)
			if err != nil {
				t.Fatalf("short-circuit run error: %v", err)
			}
			if res3.ExitCode == 0 {
				t.Errorf("short-circuit command exited 0, want nonzero")
			}
			if strings.Contains(res3.Stdout, "RAN") {
				t.Errorf("short-circuit command ran echo RAN despite an earlier `false`; stdout=%q", res3.Stdout)
			}
		})
	}
}

func TestClients_BuildCmdMatchesRepo(t *testing.T) {
	// Assert that for EVERY client, BuildCmd contains the host+path of its own Repo field.
	// BuildCmd's clone URL must start with Repo + ".git" or Repo (normalized).
	for id, c := range clients {
		id, c := id, c
		t.Run(id, func(t *testing.T) {
			// Extract the git clone URL from BuildCmd
			// The pattern is: `git clone --depth 1 <URL> <target>`
			clonePrefix := "git clone --depth 1 "
			idx := strings.Index(c.BuildCmd, clonePrefix)
			if idx == -1 {
				t.Fatalf("client %q BuildCmd does not contain 'git clone --depth 1'", id)
			}
			startIdx := idx + len(clonePrefix)
			endIdx := strings.Index(c.BuildCmd[startIdx:], " ")
			if endIdx == -1 {
				t.Fatalf("client %q BuildCmd has malformed git clone", id)
			}
			cloneURL := c.BuildCmd[startIdx : startIdx+endIdx]

			// Normalize: Repo might end with .git or not; BuildCmd clone URL might end with .git or not
			repoNorm := strings.TrimSuffix(c.Repo, ".git")
			cloneURLNorm := strings.TrimSuffix(cloneURL, ".git")

			if cloneURLNorm != repoNorm {
				t.Errorf("client %q: BuildCmd clones %q but Repo is %q", id, cloneURL, c.Repo)
			}
		})
	}
}

// ---- v0.2 Task 1: ports, data subtrees, size data ----

func TestWireConfig_PortDefaults(t *testing.T) {
	w := WireConfig{}
	if got := w.ExecHTTP(); got != 8545 {
		t.Errorf("zero-value WireConfig.ExecHTTP() = %d, want 8545", got)
	}
	if got := w.BeaconHTTP(); got != 5052 {
		t.Errorf("zero-value WireConfig.BeaconHTTP() = %d, want 5052", got)
	}
	if got := w.ExecP2P(); got != 30303 {
		t.Errorf("zero-value WireConfig.ExecP2P() = %d, want 30303", got)
	}
}

func TestWireConfig_PortResolversHonorNonZeroFields(t *testing.T) {
	w := WireConfig{ExecHTTPPort: 9545, BeaconHTTPPort: 6052, ExecP2PPort: 40303}
	if got := w.ExecHTTP(); got != 9545 {
		t.Errorf("ExecHTTP() = %d, want 9545", got)
	}
	if got := w.BeaconHTTP(); got != 6052 {
		t.Errorf("BeaconHTTP() = %d, want 6052", got)
	}
	if got := w.ExecP2P(); got != 40303 {
		t.Errorf("ExecP2P() = %d, want 40303", got)
	}
}

// beaconHTTPPortToken returns the exact rendered-unit flag token a beacon
// client's own family uses for its HTTP API port, given the resolved port
// number — every family but prysm shares "--http-port N"; prysm uses
// "--grpc-gateway-port=N".
func beaconHTTPPortToken(beaconID string, port int) string {
	if beaconID == "prysm-pulse" {
		return fmt.Sprintf("--grpc-gateway-port=%d", port)
	}
	return fmt.Sprintf("--http-port %d", port)
}

// TestRenderUnits_PortsDefaultAndCustom is a golden test, for every valid
// exec+beacon combo in the catalog, that (1) a zero-value WireConfig
// renders the documented defaults (8545/5052/30303) and (2) a WireConfig
// with all three ports set renders those exact custom values — and that
// the old default values are nowhere in the custom-ports render (proving
// the fields are actually threaded through, not just accidentally always
// matching the default).
func TestRenderUnits_PortsDefaultAndCustom(t *testing.T) {
	for _, net := range Networks() {
		net := net
		for _, execID := range net.ExecClients {
			for _, beaconID := range net.BeaconClients {
				execID, beaconID := execID, beaconID
				t.Run(net.Name+"/"+execID+"+"+beaconID, func(t *testing.T) {
					base := WireConfig{
						ChainID:  net.ChainID,
						ExecID:   execID,
						BeaconID: beaconID,
						DataDir:  "/var/lib/valve-node/dir",
						JWTPath:  "/var/lib/valve-node/dir/jwt.hex",
					}

					execUnit, beaconUnit, err := RenderUnits(base)
					if err != nil {
						t.Fatalf("RenderUnits(defaults) error: %v", err)
					}
					if !strings.Contains(execUnit, "--http.port 8545") {
						t.Errorf("default exec unit missing --http.port 8545:\n%s", execUnit)
					}
					if !strings.Contains(execUnit, "--port 30303") {
						t.Errorf("default exec unit missing --port 30303:\n%s", execUnit)
					}
					if !strings.Contains(beaconUnit, beaconHTTPPortToken(beaconID, 5052)) {
						t.Errorf("default beacon unit missing %q:\n%s", beaconHTTPPortToken(beaconID, 5052), beaconUnit)
					}

					custom := base
					custom.ExecHTTPPort = 9545
					custom.BeaconHTTPPort = 6052
					custom.ExecP2PPort = 40303
					execUnit2, beaconUnit2, err := RenderUnits(custom)
					if err != nil {
						t.Fatalf("RenderUnits(custom) error: %v", err)
					}
					if !strings.Contains(execUnit2, "--http.port 9545") {
						t.Errorf("custom exec unit missing --http.port 9545:\n%s", execUnit2)
					}
					if !strings.Contains(execUnit2, "--port 40303") {
						t.Errorf("custom exec unit missing --port 40303:\n%s", execUnit2)
					}
					if !strings.Contains(beaconUnit2, beaconHTTPPortToken(beaconID, 6052)) {
						t.Errorf("custom beacon unit missing %q:\n%s", beaconHTTPPortToken(beaconID, 6052), beaconUnit2)
					}
					if strings.Contains(execUnit2, "8545") || strings.Contains(execUnit2, "30303") {
						t.Errorf("custom exec unit still contains a default port:\n%s", execUnit2)
					}
					if strings.Contains(beaconUnit2, "5052") {
						t.Errorf("custom beacon unit still contains default port 5052:\n%s", beaconUnit2)
					}

					// engine 8551 is always literal regardless of custom ports.
					if !strings.Contains(execUnit2, "8551") || !strings.Contains(beaconUnit2, "8551") {
						t.Errorf("engine port 8551 must stay literal even with custom ports")
					}
				})
			}
		}
	}
}

// TestClients_DataSubdirsAgreeWithRenderedUnits is the per-client
// template/DataSubdirs agreement test called for in task-1-brief.md: every
// client's DataSubdirs must be non-empty, and the rendered unit must
// either point its --datadir flag directly at the owned subdir (the
// prysm/lighthouse families, which own exactly one subtree) or use a bare
// --datadir <DataDir> that leaves the subtree(s) as implicit children
// (reth/erigon/geth-family, which each own one or more subtrees the
// client itself creates under a shared datadir).
func TestClients_DataSubdirsAgreeWithRenderedUnits(t *testing.T) {
	cases := []struct {
		clientID string
		w        WireConfig
		isBeacon bool
	}{
		{"reth", WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, false},
		{"geth", WireConfig{ChainID: 1, ExecID: "geth", BeaconID: "lighthouse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, false},
		{"go-pulse", WireConfig{ChainID: 369, ExecID: "go-pulse", BeaconID: "lighthouse-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, false},
		{"erigon-pulse", WireConfig{ChainID: 369, ExecID: "erigon-pulse", BeaconID: "lighthouse-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, false},
		{"lighthouse-pulse", WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, true},
		{"prysm-pulse", WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "prysm-pulse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, true},
		{"lighthouse", WireConfig{ChainID: 1, ExecID: "reth", BeaconID: "lighthouse", DataDir: "/data", JWTPath: "/data/jwt.hex"}, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.clientID, func(t *testing.T) {
			c, ok := ClientByID(tc.clientID)
			if !ok {
				t.Fatalf("ClientByID(%q) not found", tc.clientID)
			}
			if len(c.DataSubdirs) == 0 {
				t.Fatalf("client %q has empty DataSubdirs", tc.clientID)
			}

			execUnit, beaconUnit, err := RenderUnits(tc.w)
			if err != nil {
				t.Fatalf("RenderUnits: %v", err)
			}
			unit := execUnit
			if tc.isBeacon {
				unit = beaconUnit
			}

			if tc.isBeacon {
				// Single owned subtree: the datadir flag must point
				// directly at DataDir/<subdir>.
				want := tc.w.DataDir + "/" + c.DataSubdirs[0]
				if !strings.Contains(unit, want) {
					t.Errorf("client %q unit does not point --datadir at owned subtree %q:\n%s", tc.clientID, want, unit)
				}
				return
			}

			// Exec family: a bare --datadir <DataDir> (not redirected into
			// any owned subtree — the subtree(s) are implicit children the
			// client itself creates).
			bareDatadir := "--datadir " + tc.w.DataDir + " "
			if !strings.Contains(unit+" ", bareDatadir) {
				t.Errorf("client %q unit does not use bare %q (implicit subtree):\n%s", tc.clientID, bareDatadir, unit)
			}
			for _, sub := range c.DataSubdirs {
				bad := "--datadir " + tc.w.DataDir + "/" + sub
				if strings.Contains(unit, bad) {
					t.Errorf("client %q unit unexpectedly redirects --datadir into owned subtree %q (should be implicit):\n%s", tc.clientID, bad, unit)
				}
			}
		})
	}
}

// TestNetworks_SizeAndSyncLabels locks in the size/label data ported
// verbatim from packages/web/src/learn/data/networks.ts, per the interface
// block in task-1-brief.md.
func TestNetworks_SizeAndSyncLabels(t *testing.T) {
	cases := []struct {
		chainID          int
		archiveSizeTB    float64
		syncLabel        string
		genesisSyncLabel string
	}{
		{369, 3.9, "< 1 day (snapshot)", "~8 days (genesis)"},
		{943, 1.2, "~4 hrs (snapshot)", "~2 days (genesis)"},
		{1, 3.6, "~12 hrs (snapshot)", "~10 days (genesis)"},
	}
	for _, tc := range cases {
		net, ok := NetworkByChainID(tc.chainID)
		if !ok {
			t.Fatalf("NetworkByChainID(%d) not found", tc.chainID)
		}
		if net.ArchiveSizeTB != tc.archiveSizeTB {
			t.Errorf("chain %d ArchiveSizeTB = %v, want %v", tc.chainID, net.ArchiveSizeTB, tc.archiveSizeTB)
		}
		if net.SyncLabel != tc.syncLabel {
			t.Errorf("chain %d SyncLabel = %q, want %q", tc.chainID, net.SyncLabel, tc.syncLabel)
		}
		if net.GenesisSyncLabel != tc.genesisSyncLabel {
			t.Errorf("chain %d GenesisSyncLabel = %q, want %q", tc.chainID, net.GenesisSyncLabel, tc.genesisSyncLabel)
		}
	}
}

// TestExpectedBytes_MatchesSpec locks in ExpectedBytes' archive and full
// (half-archive) figures as exact byte counts, ported verbatim from
// packages/web/src/learn/data/networks.ts's snapshot.sizeTB.
func TestExpectedBytes_MatchesSpec(t *testing.T) {
	cases := []struct {
		chainID   int
		archive   bool
		wantBytes uint64
	}{
		{369, true, 3_900_000_000_000},
		{369, false, 1_950_000_000_000},
		{943, true, 1_200_000_000_000},
		{943, false, 600_000_000_000},
		{1, true, 3_600_000_000_000},
		{1, false, 1_800_000_000_000},
	}
	for _, tc := range cases {
		got, err := ExpectedBytes(tc.chainID, tc.archive)
		if err != nil {
			t.Fatalf("ExpectedBytes(%d, %v) error: %v", tc.chainID, tc.archive, err)
		}
		if got != tc.wantBytes {
			t.Errorf("ExpectedBytes(%d, %v) = %d, want %d", tc.chainID, tc.archive, got, tc.wantBytes)
		}
	}
}

func TestExpectedBytes_UnknownChainErrors(t *testing.T) {
	if _, err := ExpectedBytes(9999, true); err == nil {
		t.Fatal("ExpectedBytes(9999, true) expected error for unknown chain id, got nil")
	}
}

// TestRenderUnits_PulseNetworkSelectorFlags is a golden-value regression
// test for the go-pulse and prysm-pulse network-selector flags, verified
// E2E against the actual built binaries' --help on a live box:
//
//	go-pulse:   --pulsechain (369) / --pulsechain-testnet-v4 (943)
//	prysm-pulse: --pulsechain (369) / --pulsechain-testnet-v4 (943)
//
// The previous invented forms (--pulsechain.testnet, --pulsechain-testnet)
// are rejected by the binaries with "flag provided but not defined".
func TestRenderUnits_PulseNetworkSelectorFlags(t *testing.T) {
	// badTestnetFlag matches the old, wrong --pulsechain-testnet flag as an
	// exact token (not as a substring prefix of --pulsechain-testnet-v4).
	badTestnetFlag := regexp.MustCompile(`--pulsechain-testnet(\s|$)`)

	cases := []struct {
		name        string
		w           WireConfig
		wantExec    []string
		wantBeacon  []string
		notWantExec []string
		notWantBcn  []string
	}{
		{
			name: "943 go-pulse + prysm-pulse",
			w: WireConfig{
				ChainID: 943, ExecID: "go-pulse", BeaconID: "prysm-pulse",
				DataDir: "/var/lib/valve-node/943",
			},
			wantExec:   []string{"--pulsechain-testnet-v4"},
			wantBeacon: []string{"--pulsechain-testnet-v4"},
		},
		{
			name: "369 go-pulse + prysm-pulse",
			w: WireConfig{
				ChainID: 369, ExecID: "go-pulse", BeaconID: "prysm-pulse",
				DataDir: "/var/lib/valve-node/369",
			},
			wantExec:   []string{"--pulsechain"},
			wantBeacon: []string{"--pulsechain"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			execUnit, beaconUnit, err := RenderUnits(tc.w)
			if err != nil {
				t.Fatalf("RenderUnits: %v", err)
			}
			for _, want := range tc.wantExec {
				if !strings.Contains(execUnit, want) {
					t.Errorf("exec unit missing %q\n%s", want, execUnit)
				}
			}
			for _, want := range tc.wantBeacon {
				if !strings.Contains(beaconUnit, want) {
					t.Errorf("beacon unit missing %q\n%s", want, beaconUnit)
				}
			}
			if strings.Contains(execUnit, "--pulsechain.testnet") {
				t.Errorf("exec unit contains invented flag --pulsechain.testnet\n%s", execUnit)
			}
			if badTestnetFlag.MatchString(execUnit) {
				t.Errorf("exec unit contains invented flag --pulsechain-testnet (exact token)\n%s", execUnit)
			}
			if badTestnetFlag.MatchString(beaconUnit) {
				t.Errorf("beacon unit contains invented flag --pulsechain-testnet (exact token)\n%s", beaconUnit)
			}
		})
	}
}
