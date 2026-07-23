package setup

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// neededToolchains returns the deduplicated, sorted union of
// catalog.Client.Toolchain across clients — the set of build toolchains
// the toolchain step must ensure are present before install runs
// BuildCmd. Clients with no Toolchain set (should not occur in the
// catalog, but defensive) are skipped.
func neededToolchains(clients ...catalog.Client) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range clients {
		if c.Toolchain == "" || seen[c.Toolchain] {
			continue
		}
		seen[c.Toolchain] = true
		out = append(out, c.Toolchain)
	}
	sort.Strings(out)
	return out
}

// rustupInstallCmd installs a minimal rustup-managed rust toolchain
// non-interactively. It never touches apt — rustup's install script is
// self-contained.
const rustupInstallCmd = `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal`

// cargoEnvPrefix sources a rustup-installed cargo's env file if present,
// so a fresh shell (which hasn't sourced ~/.profile/~/.bashrc since
// rustup install) still resolves `cargo` on PATH. Matches the prefix
// catalog's rust-toolchain BuildCmds use.
const cargoEnvPrefix = `. "$HOME/.cargo/env" 2>/dev/null || true; `

// needsCC reports whether any toolchain in `needed` requires a working C
// compiler: "go" needs it for cgo (prysm-pulse's herumi-bls/blst — without
// cc, Go silently sets CGO_ENABLED=0 and the build fails with "undefined:
// SecretKey/Fr/G1..." deep into the build instead of a clear preflight
// error), and "rust" needs cc as the linker driver rustc invokes.
func needsCC(needed []string) bool {
	for _, tc := range needed {
		if tc == "go" || tc == "rust" {
			return true
		}
	}
	return false
}

// toolchainStep ensures git, a C compiler (when any needed toolchain
// requires one), and the union of build toolchains `needed` requires are
// present on the target before install runs each client's BuildCmd. It
// sits between preflight and install-exec in Plan().
//
// git and cc are checked and installed independently of each other — NOT
// bundled into a single "if git missing, apt-get git+cc" branch. On boxes
// where git pre-exists (colima/lima images, many cloud images) but cc
// does not, a git-gated cc install would silently skip cc, breaking any
// cgo- or rust-toolchain build later with no clear signal at this step.
//
// v1 targets Debian/Ubuntu only: any package install goes through apt-get.
// If apt-get itself is missing and something is actually missing (git,
// cc, or go), Run fails immediately with a clear error rather than a raw
// shell "command not found" — rust's toolchain (rustup) never needs apt,
// so a rust-only pair works even off Debian/Ubuntu as long as git and cc
// are already present.
func toolchainStep(needed []string) Step {
	return Step{
		ID:    "toolchain",
		Title: "Ensure git + toolchains (" + strings.Join(needed, ", ") + ")",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			opts := streamOpts(ctx, st, "toolchain")

			hasGit, err := commandExists(ctx, e, "git")
			if err != nil {
				return fmt.Errorf("toolchain: probe git: %w", err)
			}
			if !hasGit {
				if err := requireAPT(ctx, e); err != nil {
					return err
				}
				if err := aptInstall(ctx, e, opts, "git", "ca-certificates"); err != nil {
					return err
				}
			}

			if needsCC(needed) {
				hasCC, err := commandExists(ctx, e, "cc")
				if err != nil {
					return fmt.Errorf("toolchain: probe cc: %w", err)
				}
				if !hasCC {
					if err := requireAPT(ctx, e); err != nil {
						return err
					}
					if err := aptInstall(ctx, e, opts, "build-essential"); err != nil {
						return err
					}
				}
			}

			for _, tc := range needed {
				switch tc {
				case "go":
					hasGo, err := commandExists(ctx, e, "go")
					if err != nil {
						return fmt.Errorf("toolchain: probe go: %w", err)
					}
					if !hasGo {
						if err := requireAPT(ctx, e); err != nil {
							return err
						}
						if err := aptInstall(ctx, e, opts, "golang-go"); err != nil {
							return err
						}
					}
				case "rust":
					hasCargo, err := cargoAvailable(ctx, e)
					if err != nil {
						return fmt.Errorf("toolchain: probe cargo: %w", err)
					}
					if !hasCargo {
						res, err := e.Run(ctx, rustupInstallCmd, opts)
						if err != nil {
							return fmt.Errorf("toolchain: install rust: %w", err)
						}
						if res.ExitCode != 0 {
							return fmt.Errorf("toolchain: install rust failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
						}
					}
				default:
					return fmt.Errorf("toolchain: unknown toolchain %q", tc)
				}
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			return verifyToolchains(ctx, e, needed)
		},
	}
}

// commandExists probes whether `name` resolves on the target's PATH.
func commandExists(ctx context.Context, e executor.Executor, name string) (bool, error) {
	res, err := e.Run(ctx, "command -v "+name+" >/dev/null", nil)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// cargoAvailable probes whether cargo is either already on PATH or sitting
// at the standard rustup install location — the same either/or check the
// setup runbook uses so a prior rustup install (whose shell profile edits
// haven't been sourced yet) still counts as "present".
func cargoAvailable(ctx context.Context, e executor.Executor) (bool, error) {
	res, err := e.Run(ctx, `command -v cargo >/dev/null || [ -x "$HOME/.cargo/bin/cargo" ]`, nil)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// requireAPT fails with a clear, actionable error if apt-get is not on the
// target — the only package manager v1 knows how to drive.
func requireAPT(ctx context.Context, e executor.Executor) error {
	has, err := commandExists(ctx, e, "apt-get")
	if err != nil {
		return fmt.Errorf("toolchain: probe apt-get: %w", err)
	}
	if !has {
		return fmt.Errorf("toolchain: apt-get not found on target — v1 supports Debian/Ubuntu targets only")
	}
	return nil
}

// aptInstall runs a non-interactive apt-get update + install for pkgs.
func aptInstall(ctx context.Context, e executor.Executor, opts *executor.RunOpts, pkgs ...string) error {
	cmd := "apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends " + strings.Join(pkgs, " ")
	res, err := e.Run(ctx, cmd, opts)
	if err != nil {
		return fmt.Errorf("toolchain: apt-get install %s: %w", strings.Join(pkgs, " "), err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("toolchain: apt-get install %s failed (exit %d): %s", strings.Join(pkgs, " "), res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

// verifyToolchains is the toolchain step's marker probe: git, cc (when
// any needed toolchain requires one), and every toolchain in `needed`
// must actually run, not just resolve on PATH.
func verifyToolchains(ctx context.Context, e executor.Executor, needed []string) error {
	res, err := e.Run(ctx, "git --version", nil)
	if err != nil {
		return fmt.Errorf("toolchain: verify git: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("toolchain: git not available yet")
	}

	if needsCC(needed) {
		res, err := e.Run(ctx, "cc --version", nil)
		if err != nil {
			return fmt.Errorf("toolchain: verify cc: %w", err)
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("toolchain: cc not available yet")
		}
	}

	for _, tc := range needed {
		switch tc {
		case "go":
			res, err := e.Run(ctx, "go version", nil)
			if err != nil {
				return fmt.Errorf("toolchain: verify go: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("toolchain: go not available yet")
			}
		case "rust":
			res, err := e.Run(ctx, cargoEnvPrefix+"cargo --version", nil)
			if err != nil {
				return fmt.Errorf("toolchain: verify cargo: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("toolchain: cargo not available yet")
			}
		default:
			return fmt.Errorf("toolchain: unknown toolchain %q", tc)
		}
	}
	return nil
}
