package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

func TestLoadMissingReturnsZeroValueWithDefaultRefRPCBase(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Targets) != 0 {
		t.Errorf("Targets = %+v, want empty", c.Targets)
	}
	if c.AIProvider != "" {
		t.Errorf("AIProvider = %q, want empty", c.AIProvider)
	}
	if c.RefRPCBase != defaultRefRPCBase {
		t.Errorf("RefRPCBase = %q, want default %q", c.RefRPCBase, defaultRefRPCBase)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := Config{
		Targets: []Target{
			{ID: "local", Mode: "local"},
			{
				ID:   "box1",
				Mode: "ssh",
				SSH: &executor.SSHConfig{
					Host:        "1.2.3.4",
					User:        "root",
					KeyPath:     "/home/me/.ssh/id_ed25519",
					HostKeyFile: "/home/me/.valve-node/known_hosts",
					Port:        2222,
				},
				Wire: &catalog.WireConfig{
					ChainID:  369,
					ExecID:   "reth",
					BeaconID: "lighthouse-pulse",
					DataDir:  "/var/lib/valve-node/369",
					JWTPath:  "/var/lib/valve-node/369/jwt.hex",
					Archive:  true,
				},
			},
		},
		AIProvider: "gemini",
		AIKey:      "secret-key",
		RefRPCBase: "https://rpc.valve.city/v1/vk_custom",
	}

	if err := want.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got.Targets) != 2 {
		t.Fatalf("Targets = %+v, want 2 entries", got.Targets)
	}
	if got.Targets[0].ID != "local" || got.Targets[0].Mode != "local" {
		t.Errorf("Targets[0] = %+v, want local/local", got.Targets[0])
	}
	if got.Targets[1].SSH == nil || got.Targets[1].SSH.Host != "1.2.3.4" || got.Targets[1].SSH.Port != 2222 {
		t.Errorf("Targets[1].SSH = %+v, want host 1.2.3.4 port 2222", got.Targets[1].SSH)
	}
	if got.Targets[1].Wire == nil || got.Targets[1].Wire.ChainID != 369 || got.Targets[1].Wire.ExecID != "reth" {
		t.Errorf("Targets[1].Wire = %+v, want chain 369 exec reth", got.Targets[1].Wire)
	}
	if got.AIProvider != "gemini" || got.AIKey != "secret-key" {
		t.Errorf("AIProvider/AIKey = %q/%q, want gemini/secret-key", got.AIProvider, got.AIKey)
	}
	if got.RefRPCBase != "https://rpc.valve.city/v1/vk_custom" {
		t.Errorf("RefRPCBase = %q, want the saved override", got.RefRPCBase)
	}
}

func TestSaveWritesMode0600(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := (Config{}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(home, ".valve-node", "config.json"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config.json mode = %o, want 0600", perm)
	}
}

func TestSaveIsAtomicNoLeftoverTempFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := (Config{AIProvider: "groq"}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(home, ".valve-node"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("dir entries = %v, want exactly [config.json]", names)
	}
}

func TestDirIsHomeDotValveNode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	d, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if d != filepath.Join(home, ".valve-node") {
		t.Errorf("Dir() = %q, want %q", d, filepath.Join(home, ".valve-node"))
	}
}
