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
