package catalog

import (
	"strings"
	"testing"
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
