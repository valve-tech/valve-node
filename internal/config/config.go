// Package config persists valve-node's own local state — the targets it
// knows how to manage, and the AI provider it's configured to use for log
// explanations — to a single JSON file under the user's home directory. It
// performs no validation of the domain data it stores (that's the caller's
// job); it only knows how to read and write the file safely.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// defaultRefRPCBase is the public demo-key reference RPC base URL, used
// whenever a Config has no explicit override. Callers append "/evm/<chainId>"
// to get the per-chain reference endpoint.
const defaultRefRPCBase = "https://rpc.valve.city/v1/vk_Et-4emAlBIym1PjiCogh5p7IuGtS-Rpj"

// Target is one machine valve-node can set up and monitor a node on.
type Target struct {
	ID   string              `json:"id"`   // "local" or a slug of the host
	Mode string              `json:"mode"` // "local" | "ssh"
	SSH  *executor.SSHConfig `json:"ssh,omitempty"`
	Wire *catalog.WireConfig `json:"wire,omitempty"` // set once the wizard has run
}

// Config is valve-node's persisted local state.
type Config struct {
	Targets    []Target `json:"targets"`
	AIProvider string   `json:"aiProvider"` // ""|gemini|groq|ollama
	AIKey      string   `json:"aiKey"`
	RefRPCBase string   `json:"refRpcBase"` // default: defaultRefRPCBase
}

// configFileName is the file Load/Save read and write inside Dir().
const configFileName = "config.json"

// Dir returns the directory valve-node's local state lives in
// (~/.valve-node), without creating it.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".valve-node"), nil
}

func filePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads Config from ~/.valve-node/config.json. A missing file is not an
// error: it returns the zero Config (with RefRPCBase defaulted). RefRPCBase
// is defaulted whenever it's empty, whether that's because the file doesn't
// exist yet or because a stored config happens to have it blank.
func Load() (Config, error) {
	path, err := filePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{RefRPCBase: defaultRefRPCBase}, nil
		}
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if c.RefRPCBase == "" {
		c.RefRPCBase = defaultRefRPCBase
	}
	return c, nil
}

// Save writes c to ~/.valve-node/config.json, creating the directory if
// needed. The write is atomic (write to a temp file in the same directory,
// then rename over the target) and the file is mode 0600, since it may
// contain an AI provider API key.
func (c Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}

	path := filepath.Join(dir, configFileName)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	tmp, err := os.CreateTemp(dir, configFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// If anything below fails before the rename, don't leave the temp file
	// behind.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("config: chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("config: rename into place: %w", err)
	}
	success = true
	return nil
}
