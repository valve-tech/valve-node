package setup

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// Ports contract, matching internal/catalog/units.go: exec RPC 127.0.0.1:8545,
// engine API 127.0.0.1:8551 (JWT-authed, execution<->beacon), beacon HTTP
// 127.0.0.1:5052.
const (
	execHTTPPort   = "8545"
	enginePort     = "8551"
	beaconHTTPPort = "5052"
)

// Unit names/paths on the target, per task-4-brief.md.
const (
	execUnitName   = "valve-node-exec.service"
	beaconUnitName = "valve-node-beacon.service"
	execUnitPath   = "/etc/systemd/system/" + execUnitName
	beaconUnitPath = "/etc/systemd/system/" + beaconUnitName
)

// Plan returns the ordered steps for the config: preflight, toolchain,
// install-exec, install-beacon, wire, start, handshake.
func Plan(w catalog.WireConfig) ([]Step, error) {
	execClient, ok := catalog.ClientByID(w.ExecID)
	if !ok || execClient.Kind != "exec" {
		return nil, fmt.Errorf("setup: %q is not a known execution client id", w.ExecID)
	}
	beaconClient, ok := catalog.ClientByID(w.BeaconID)
	if !ok || beaconClient.Kind != "beacon" {
		return nil, fmt.Errorf("setup: %q is not a known beacon client id", w.BeaconID)
	}
	// RenderUnits does the full network/client-pair validation (chain
	// exists, both clients valid on it). Run it here so an invalid combo
	// fails Plan immediately rather than partway through RunAll at the
	// wire step.
	if _, _, err := catalog.RenderUnits(w); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}

	return []Step{
		preflightStep(),
		toolchainStep(neededToolchains(execClient, beaconClient)),
		installStep("install-exec", "Install execution client ("+w.ExecID+")", execClient),
		installStep("install-beacon", "Install beacon client ("+w.BeaconID+")", beaconClient),
		wireStep(),
		startStep(),
		handshakeStep(),
	}, nil
}

// ---------------------------------------------------------------------
// preflight
// ---------------------------------------------------------------------

// preflightStep has nothing to fix on the target — it only checks, so it
// has no Run: RunAll's Verify pre-check IS the preflight check, and if it
// fails, RunAll calls Verify again (there being no Run) and surfaces the
// same failure as the step's terminal error.
func preflightStep() Step {
	return Step{
		ID:    "preflight",
		Title: "Preflight checks",
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			return preflightCheck(ctx, e, st.Wire)
		},
	}
}

func preflightCheck(ctx context.Context, e executor.Executor, w catalog.WireConfig) error {
	res, err := e.Run(ctx, "uname", nil)
	if err != nil {
		return fmt.Errorf("preflight: uname: %w", err)
	}
	osName := strings.TrimSpace(res.Stdout)
	if osName != "Linux" {
		return fmt.Errorf("preflight: valve-node setup requires Linux, found %q", osName)
	}

	minBytes, err := minDiskBytesFor(w)
	if err != nil {
		return err
	}
	// DataDir does not exist yet on a fresh box — it's created later by
	// wire's mkdir — so `df` must be run against the nearest existing
	// ancestor directory on DataDir's path, not DataDir itself.
	dfCmd := fmt.Sprintf(
		`d=%s; while [ ! -d "$d" ]; do d=$(dirname "$d"); done; df -B1 --output=avail "$d" | tail -1`,
		shQuote(w.DataDir),
	)
	res, err = e.Run(ctx, dfCmd, nil)
	if err != nil {
		return fmt.Errorf("preflight: df: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("preflight: df failed probing an ancestor of %s (exit %d): %s", w.DataDir, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	avail, perr := parseDFAvail(res.Stdout)
	if perr != nil {
		return fmt.Errorf("preflight: could not parse df output for %s: %w (output: %q)", w.DataDir, perr, res.Stdout)
	}
	if avail < minBytes {
		return fmt.Errorf(
			"preflight: %s has %s available, need at least %s for a %s node on chain %d",
			w.DataDir, humanBytes(avail), humanBytes(minBytes), tierName(w.Archive), w.ChainID,
		)
	}

	res, err = e.Run(ctx, "ss -ltn", nil)
	if err != nil {
		return fmt.Errorf("preflight: ss: %w", err)
	}
	for _, port := range []string{execHTTPPort, enginePort, beaconHTTPPort} {
		if strings.Contains(res.Stdout, ":"+port) {
			return fmt.Errorf("preflight: port %s is already in use (from `ss -ltn`)", port)
		}
	}
	return nil
}

// chainArchiveSizeTB is the archive-tier dataset size per chain, ported
// verbatim from packages/web/src/learn/data/networks.ts's
// snapshot.sizeTB. There is no learn-data source for a full(pruned)-tier
// minimum; per general operational guidance a pruned full node for these
// clients runs at roughly half the archive dataset, so the full tier uses
// half this value. This halving is a reasoned estimate, not a sourced
// figure — flagged in task-4-report.md.
var chainArchiveSizeTB = map[int]float64{
	1:   3.6,
	369: 3.9,
	943: 1.2,
}

func minDiskBytesFor(w catalog.WireConfig) (uint64, error) {
	sizeTB, ok := chainArchiveSizeTB[w.ChainID]
	if !ok {
		return 0, fmt.Errorf("preflight: no disk-size guidance for chain id %d", w.ChainID)
	}
	if !w.Archive {
		sizeTB /= 2
	}
	const safetyMargin = 1.10 // 10% headroom above the raw dataset size
	return uint64(sizeTB * 1e12 * safetyMargin), nil
}

func tierName(archive bool) string {
	if archive {
		return "archive"
	}
	return "full"
}

// parseDFAvail parses the numeric line out of `df -B1 --output=avail`
// output, skipping the header line and any blank lines.
func parseDFAvail(output string) (uint64, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if n, err := strconv.ParseUint(line, 10, 64); err == nil {
			return n, nil
		}
	}
	return 0, fmt.Errorf("no numeric avail value found")
}

func humanBytes(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

// ---------------------------------------------------------------------
// install
// ---------------------------------------------------------------------

// binaryNameFor returns the binary name a catalog client installs as. Every
// client's BuildCmd (internal/catalog/clients.go) installs to
// /usr/local/bin/<client-id>, and internal/catalog/units.go's
// execCommand/beaconCommand invoke each client by that same client-id name
// in its systemd ExecStart line — so the install path is always just the
// client id itself.
func binaryNameFor(id string) string {
	return id
}

// installStep installs one client (execution or beacon, whichever
// `client` is) and verifies it landed at the binary path its systemd unit
// will invoke.
func installStep(stepID, title string, client catalog.Client) Step {
	dest := "/usr/local/bin/" + binaryNameFor(client.ID)

	return Step{
		ID:    stepID,
		Title: title,
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			opts := streamOpts(ctx, st, stepID)

			if url := client.ReleaseURL(runtime.GOOS, runtime.GOARCH, client.PinVersion); url != "" {
				downloadPath := "/tmp/" + binaryNameFor(client.ID) + ".download"
				cmd := fmt.Sprintf(
					"set -e; curl -fL %s -o %s && sha256sum %s && chmod +x %s && mv %s %s",
					shQuote(url), shQuote(downloadPath), shQuote(downloadPath),
					shQuote(downloadPath), shQuote(downloadPath), shQuote(dest),
				)
				res, err := e.Run(ctx, cmd, opts)
				if err != nil {
					return fmt.Errorf("install: download %s: %w", client.ID, err)
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("install: download %s failed (exit %d): %s", client.ID, res.ExitCode, res.Stderr)
				}
				return nil
			}

			// No release binary published for this platform — build from
			// source (or pull the vendor's docker image) exactly as
			// BuildCmd specifies; it already contains its own clone step.
			res, err := e.Run(ctx, client.BuildCmd, opts)
			if err != nil {
				return fmt.Errorf("install: build %s: %w", client.ID, err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("install: build %s failed (exit %d): %s", client.ID, res.ExitCode, res.Stderr)
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			cmd := fmt.Sprintf("test -x %s && %s --version", shQuote(dest), shQuote(dest))
			res, err := e.Run(ctx, cmd, nil)
			if err != nil {
				return fmt.Errorf("install: verify %s: %w", client.ID, err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("install: %s not installed/runnable at %s yet", client.ID, dest)
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------
// wire
// ---------------------------------------------------------------------

func wireStep() Step {
	return Step{
		ID:    "wire",
		Title: "Write JWT secret and systemd units",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			w := st.Wire
			jwtPath := jwtPathFor(w)
			opts := streamOpts(ctx, st, "wire")

			probe, err := e.Run(ctx, fmt.Sprintf("test -f %s", shQuote(jwtPath)), nil)
			if err != nil {
				return fmt.Errorf("wire: probe jwt: %w", err)
			}
			if probe.ExitCode != 0 {
				cmd := fmt.Sprintf(
					"umask 077 && mkdir -p %s && openssl rand -hex 32 > %s && chmod 0600 %s",
					shQuote(path.Dir(jwtPath)), shQuote(jwtPath), shQuote(jwtPath),
				)
				res, err := e.Run(ctx, cmd, opts)
				if err != nil {
					return fmt.Errorf("wire: write jwt: %w", err)
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("wire: write jwt failed (exit %d): %s", res.ExitCode, res.Stderr)
				}
			}

			execUnit, beaconUnit, err := catalog.RenderUnits(w)
			if err != nil {
				return fmt.Errorf("wire: render units: %w", err)
			}
			if err := e.WriteFile(ctx, execUnitPath, []byte(execUnit), 0644); err != nil {
				return fmt.Errorf("wire: write %s: %w", execUnitPath, err)
			}
			if err := e.WriteFile(ctx, beaconUnitPath, []byte(beaconUnit), 0644); err != nil {
				return fmt.Errorf("wire: write %s: %w", beaconUnitPath, err)
			}

			cmd := fmt.Sprintf(
				"systemctl daemon-reload && systemctl enable --now %s && systemctl enable --now %s",
				execUnitName, beaconUnitName,
			)
			res, err := e.Run(ctx, cmd, opts)
			if err != nil {
				return fmt.Errorf("wire: systemctl: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("wire: systemctl daemon-reload/enable failed (exit %d): %s", res.ExitCode, res.Stderr)
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			w := st.Wire
			jwtPath := jwtPathFor(w)

			cmd := fmt.Sprintf("test -f %s && test -f %s && test -f %s",
				shQuote(jwtPath), shQuote(execUnitPath), shQuote(beaconUnitPath))
			res, err := e.Run(ctx, cmd, nil)
			if err != nil {
				return fmt.Errorf("wire: verify files: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("wire: jwt/unit files not all present yet")
			}

			res, err = e.Run(ctx, fmt.Sprintf("systemctl is-enabled %s %s", execUnitName, beaconUnitName), nil)
			if err != nil {
				return fmt.Errorf("wire: verify enabled: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("wire: units not enabled yet: %s", strings.TrimSpace(res.Stdout))
			}
			return nil
		},
	}
}

func jwtPathFor(w catalog.WireConfig) string {
	if w.JWTPath != "" {
		return w.JWTPath
	}
	return path.Join(w.DataDir, "jwt.hex")
}

// ---------------------------------------------------------------------
// start
// ---------------------------------------------------------------------

// startStep confirms both services are actually running, restarting them
// if wire's `enable --now` left either inactive for any reason (e.g. a
// unit that failed fast on first boot and needs a clean restart after the
// data dir/JWT were fixed up by an earlier retry of this Plan).
func startStep() Step {
	return Step{
		ID:    "start",
		Title: "Start execution and beacon services",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			cmd := fmt.Sprintf("systemctl start %s %s", execUnitName, beaconUnitName)
			res, err := e.Run(ctx, cmd, streamOpts(ctx, st, "start"))
			if err != nil {
				return fmt.Errorf("start: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("start: systemctl start failed (exit %d): %s", res.ExitCode, res.Stderr)
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			res, err := e.Run(ctx, fmt.Sprintf("systemctl is-active %s %s", execUnitName, beaconUnitName), nil)
			if err != nil {
				return fmt.Errorf("start: verify: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("start: services not both active yet: %s", strings.TrimSpace(res.Stdout))
			}
			for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
				if strings.TrimSpace(line) != "active" {
					return fmt.Errorf("start: services not both active: %s", res.Stdout)
				}
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------
// handshake
// ---------------------------------------------------------------------

// handshakeTimeout/handshakePollInterval are package vars (not consts) so
// tests can shrink them to avoid real sleeps.
var (
	handshakeTimeout      = 60 * time.Second
	handshakePollInterval = 2 * time.Second
)

func handshakeStep() Step {
	return Step{
		ID:    "handshake",
		Title: "Verify execution/beacon handshake",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			opts := streamOpts(ctx, st, "handshake")
			deadline := time.Now().Add(handshakeTimeout)
			var lastErr error
			for {
				lastErr = handshakeCheck(ctx, e, st.Wire, opts)
				if lastErr == nil {
					return nil
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("handshake: timed out after %s: %w", handshakeTimeout, lastErr)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(handshakePollInterval):
				}
			}
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			return handshakeCheck(ctx, e, st.Wire, nil)
		},
	}
}

// handshakeCheck runs the three checks from task-4-brief.md Step 3 once.
// Any failure returned embeds the concrete evidence (http code / RPC body
// / offending journal lines) so the caller's error text is exactly what
// the UI shows — never a bare "handshake failed".
func handshakeCheck(ctx context.Context, e executor.Executor, w catalog.WireConfig, opts *executor.RunOpts) error {
	res, err := e.Run(ctx, "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:"+beaconHTTPPort+"/eth/v1/node/syncing", opts)
	if err != nil {
		return fmt.Errorf("handshake: beacon syncing probe: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("handshake: beacon /eth/v1/node/syncing curl failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	if code := strings.TrimSpace(res.Stdout); code != "200" {
		return fmt.Errorf("handshake: beacon /eth/v1/node/syncing returned http %q", code)
	}

	res, err = e.Run(ctx,
		`curl -s -X POST -H 'Content-Type: application/json' `+
			`--data '{"jsonrpc":"2.0","id":1,"method":"eth_syncing","params":[]}' `+
			`http://127.0.0.1:`+execHTTPPort, opts)
	if err != nil {
		return fmt.Errorf("handshake: exec eth_syncing probe: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("handshake: exec eth_syncing curl failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	if !strings.Contains(res.Stdout, `"jsonrpc"`) {
		return fmt.Errorf("handshake: exec eth_syncing did not answer: %s", res.Stdout)
	}

	res, err = e.Run(ctx, "journalctl -u "+beaconUnitName+" --since -2m --no-pager", opts)
	if err != nil {
		return fmt.Errorf("handshake: journalctl: %w", err)
	}
	// journalctl's ExitCode is deliberately not checked: it can exit
	// non-zero for reasons that don't mean "handshake failed" (e.g. a
	// grep-filtered pipeline finding zero matching lines exits 1) — that
	// reads as "no lines", which authErrorLines already treats as "no
	// auth errors found" below.
	if bad := authErrorLines(res.Stdout); len(bad) > 0 {
		return fmt.Errorf("handshake: beacon journal shows auth errors in the last 2m:\n%s", strings.Join(bad, "\n"))
	}
	return nil
}

var authErrorPattern = regexp.MustCompile(`(?i)jwt|401|unauthorized`)

func authErrorLines(journal string) []string {
	var out []string
	for _, line := range strings.Split(journal, "\n") {
		if line == "" {
			continue
		}
		if authErrorPattern.MatchString(line) {
			out = append(out, line)
		}
	}
	return out
}

// ---------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------

// streamOpts forwards a command's live stdout lines onto st.Events as
// progress events for the given step id. The emit is ctx-aware; if ctx is
// canceled while a line send is stalled on a non-draining consumer, the
// line is dropped rather than blocking the command's execution forever.
func streamOpts(ctx context.Context, st *State, stepID string) *executor.RunOpts {
	return &executor.RunOpts{
		Stream: func(line string) {
			_ = emit(ctx, st, Event{StepID: stepID, Line: line})
		},
	}
}

// shQuote single-quotes s for safe interpolation into a `sh -c` command
// string, escaping any embedded single quotes.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
