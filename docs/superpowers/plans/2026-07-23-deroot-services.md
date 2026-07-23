# De-Root Node Services Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run the execution and beacon clients as a dedicated unprivileged system user (`valve-node`) with hardened systemd units, instead of root, with migration handled automatically by re-running setup.

**Architecture:** The systemd unit template in `internal/catalog/units.go` gains `User=`/`Group=` plus a conservative hardening block (`NoNewPrivileges`, `ProtectSystem=strict` + `ReadWritePaths=<DataDir>`, `PrivateTmp`, etc., and `AmbientCapabilities=CAP_NET_BIND_SERVICE` only when a configured port is <1024). A new idempotent `account` setup step creates the system user; the existing `wire` step chowns the data tree to it after the JWT write and verifies ownership. Migration of an existing root install falls out of the existing wire-step machinery: re-running setup detects changed unit content, rewrites + restarts, and the chown re-owns the data. Setup itself still requires root (it writes units, creates users, chowns); only the long-running services drop privileges.

**Tech Stack:** Go 1.25, `text/template`, table tests with the package-local `fakeExecutor` scripted double.

## Global Constraints

- Service user and group are both named `valve-node`, exported as `catalog.ServiceUser` / `catalog.ServiceGroup`.
- Setup still requires root on the target — the preflight `id -u` gate and its exact error message are unchanged.
- All target mutations go through the `executor.Executor` interface (works identically local and SSH). Shell args quoted with the package-local `shQuote`.
- Unit names/paths unchanged: `valve-node-exec.service` / `valve-node-beacon.service` under `/etc/systemd/system`.
- Binaries stay root-owned in `/usr/local/bin` (read+exec is all the service user needs).
- No new `WireConfig` or API fields (the service user is a constant, not configuration).
- Verify with `go build ./... && go test ./...`. No web UI change is needed (UI has no root-related copy).

---

### Task 1: Harden the unit template (`User=`, hardening block, conditional net-bind capability)

**Files:**
- Modify: `internal/catalog/units.go` (template at lines 69–90, `renderUnit` at 149–155, call sites at 138/142)
- Test: `internal/catalog/catalog_test.go`

**Interfaces:**
- Produces: `catalog.ServiceUser = "valve-node"`, `catalog.ServiceGroup = "valve-node"` (exported consts, used by Tasks 2–3). Rendered units now contain `User=valve-node`, hardening directives, `ReadWritePaths=<DataDir>`, and — only when that unit's ports include one <1024 — `AmbientCapabilities=CAP_NET_BIND_SERVICE`.
- Consumes: existing `WireConfig` port resolvers `ExecHTTP()`, `BeaconHTTP()`, `ExecP2P()` (zero value = default; defaults 8545/5052/30303; engine port fixed 8551).

- [ ] **Step 1: Write the failing tests**

Append to `internal/catalog/catalog_test.go`:

```go
func TestRenderUnits_RunAsServiceUser(t *testing.T) {
	w := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node/369"}
	execUnit, beaconUnit, err := RenderUnits(w)
	if err != nil {
		t.Fatalf("RenderUnits: %v", err)
	}
	for _, unit := range []string{execUnit, beaconUnit} {
		for _, want := range []string{
			"User=" + ServiceUser,
			"Group=" + ServiceGroup,
			"NoNewPrivileges=true",
			"PrivateTmp=true",
			"PrivateDevices=true",
			"ProtectSystem=strict",
			"ProtectHome=true",
			"ReadWritePaths=/var/lib/valve-node/369",
			"ProtectKernelTunables=true",
			"ProtectControlGroups=true",
			"RestrictSUIDSGID=true",
		} {
			if !strings.Contains(unit, want) {
				t.Errorf("unit missing %q:\n%s", want, unit)
			}
		}
		if strings.Contains(unit, "AmbientCapabilities") {
			t.Errorf("default ports must not grant AmbientCapabilities:\n%s", unit)
		}
	}
}

func TestRenderUnits_PrivilegedPortGrantsNetBindCap(t *testing.T) {
	w := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		DataDir: "/var/lib/valve-node/369", ExecHTTPPort: 443}
	execUnit, beaconUnit, err := RenderUnits(w)
	if err != nil {
		t.Fatalf("RenderUnits: %v", err)
	}
	if !strings.Contains(execUnit, "AmbientCapabilities=CAP_NET_BIND_SERVICE") {
		t.Errorf("exec unit with port 443 missing AmbientCapabilities:\n%s", execUnit)
	}
	if strings.Contains(beaconUnit, "AmbientCapabilities") {
		t.Errorf("beacon unit on default ports must not gain AmbientCapabilities:\n%s", beaconUnit)
	}

	w = WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		DataDir: "/var/lib/valve-node/369", BeaconHTTPPort: 1023}
	execUnit, beaconUnit, err = RenderUnits(w)
	if err != nil {
		t.Fatalf("RenderUnits: %v", err)
	}
	if strings.Contains(execUnit, "AmbientCapabilities") {
		t.Errorf("exec unit on default ports must not gain AmbientCapabilities:\n%s", execUnit)
	}
	if !strings.Contains(beaconUnit, "AmbientCapabilities=CAP_NET_BIND_SERVICE") {
		t.Errorf("beacon unit with port 1023 missing AmbientCapabilities:\n%s", beaconUnit)
	}
}
```

(`strings` is already imported in `catalog_test.go`.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/catalog/ -run 'TestRenderUnits_RunAsServiceUser|TestRenderUnits_PrivilegedPortGrantsNetBindCap' -v`
Expected: FAIL — `ServiceUser` undefined (compile error).

- [ ] **Step 3: Implement the template change**

In `internal/catalog/units.go`, add above `unitTemplate`:

```go
// ServiceUser/ServiceGroup are the dedicated unprivileged system account
// the execution and beacon services run as (the User=/Group= lines in the
// rendered units). Setup's account step creates the account; the wire step
// chowns the data dir to it.
const (
	ServiceUser  = "valve-node"
	ServiceGroup = "valve-node"
)
```

Replace `unitTemplate`, `unitVars`, and `renderUnit` (currently lines 69–90 and 149–155):

```go
const unitTemplate = `[Unit]
Description=valve-node {{.Description}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={{.User}}
Group={{.Group}}
ExecStart={{.ExecStart}}
Restart=always
RestartSec=5
LimitNOFILE=1048576
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths={{.DataDir}}
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictSUIDSGID=true
{{- if .NetBindCap}}
AmbientCapabilities=CAP_NET_BIND_SERVICE{{end}}

[Install]
WantedBy=multi-user.target
`

type unitVars struct {
	Description string
	ExecStart   string
	User, Group string
	DataDir     string
	// NetBindCap grants CAP_NET_BIND_SERVICE when one of this unit's
	// configured ports is privileged (<1024) — without it the
	// unprivileged service user cannot bind such a port.
	NetBindCap bool
}
```

```go
func renderUnit(description, execStart string, w WireConfig, netBindCap bool) (string, error) {
	var buf bytes.Buffer
	err := unitTmpl.Execute(&buf, unitVars{
		Description: description,
		ExecStart:   execStart,
		User:        ServiceUser,
		Group:       ServiceGroup,
		DataDir:     w.DataDir,
		NetBindCap:  netBindCap,
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
```

Update the two call sites in `RenderUnits` (lines 138–145). The engine API port is always 8551 (unprivileged), so only the configurable ports matter:

```go
	execUnit, err = renderUnit("execution client ("+w.ExecID+")", execCmd, w,
		w.ExecHTTP() < 1024 || w.ExecP2P() < 1024)
	if err != nil {
		return "", "", err
	}
	beaconUnit, err = renderUnit("beacon client ("+w.BeaconID+")", beaconCmd, w,
		w.BeaconHTTP() < 1024)
	if err != nil {
		return "", "", err
	}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/catalog/ -v`
Expected: the two new tests PASS. If any existing test (e.g. `TestRenderUnits_ValidCombos`, line 59) asserts exact unit content that the new lines break, extend its expected substrings — the assertions there are `strings.Contains` on `[Unit]`/`Restart=always`/`WantedBy=multi-user.target` and should still pass unmodified; fix only genuine failures, never by weakening the new assertions.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/units.go internal/catalog/catalog_test.go
git commit -m "feat(catalog): render units with dedicated service user and systemd hardening"
```

---

### Task 2: `account` setup step — create the system user

**Files:**
- Modify: `internal/setup/steps.go` (add `accountStep` after the preflight section ~line 144; insert into `Plan` at lines 53–61)
- Test: `internal/setup/steps_test.go` (new tests; update `TestPlan_ReturnsOrderedSteps` at 795–809)

**Interfaces:**
- Consumes: `catalog.ServiceUser`, `catalog.ServiceGroup` (Task 1); existing `Step`/`State` structs (`engine.go:27–48`), `streamOpts(ctx, st, stepID)` helper, `fakeExecutor` double.
- Produces: step ID `"account"`, title `Create service account (valve-node)`, ordered between `preflight` and `toolchain`. Idempotent: Run is a no-op when the user already exists; Verify passes iff `id -u valve-node` exits 0.

- [ ] **Step 1: Write the failing tests**

Append to `internal/setup/steps_test.go`:

```go
// ---- account ----

func TestAccount_RunCreatesSystemUserIdempotently(t *testing.T) {
	e := newFakeExecutor()
	step := accountStep()
	if err := step.Run(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var cmd string
	for _, c := range e.callLog() {
		if strings.Contains(c, "useradd") {
			cmd = c
		}
	}
	if cmd == "" {
		t.Fatalf("no useradd command issued; calls: %v", e.callLog())
	}
	for _, want := range []string{
		"id -u " + catalog.ServiceUser,
		"useradd --system --user-group",
		"--shell /usr/sbin/nologin",
		"--no-create-home",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("useradd command %q missing %q", cmd, want)
		}
	}
}

func TestAccount_RunFailsWhenUseraddFails(t *testing.T) {
	e := newFakeExecutor().
		script("useradd", executor.Result{ExitCode: 1, Stderr: "useradd: cannot lock /etc/passwd"})
	step := accountStep()
	err := step.Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("want error when useradd fails, got nil")
	}
	if !strings.Contains(err.Error(), "cannot lock /etc/passwd") {
		t.Fatalf("error %q does not surface useradd stderr", err)
	}
}

func TestAccount_VerifyChecksUserExists(t *testing.T) {
	e := newFakeExecutor().
		script("id -u "+catalog.ServiceUser, executor.Result{ExitCode: 1, Stderr: "no such user"})
	step := accountStep()
	if err := step.Verify(context.Background(), e, &State{Wire: testWire()}); err == nil {
		t.Fatal("want Verify error when the service user does not exist, got nil")
	}

	e = newFakeExecutor().
		script("id -u "+catalog.ServiceUser, executor.Result{Stdout: "998\n", ExitCode: 0})
	if err := step.Verify(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("want Verify to pass when the user exists, got %v", err)
	}
}
```

Update `TestPlan_ReturnsOrderedSteps` (line 800):

```go
	want := []string{"preflight", "account", "toolchain", "install-exec", "install-beacon", "wire", "start", "handshake"}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/setup/ -run 'TestAccount|TestPlan_ReturnsOrderedSteps' -v`
Expected: FAIL — `accountStep` undefined (compile error).

- [ ] **Step 3: Implement `accountStep`**

In `internal/setup/steps.go`, insert `accountStep()` into `Plan`'s step list (line 53–61) directly after `preflightStep()`:

```go
	return []Step{
		preflightStep(),
		accountStep(),
		toolchainStep(neededToolchains(execClient, beaconClient)),
		installStep("install-exec", "Install execution client ("+w.ExecID+")", execClient),
		installStep("install-beacon", "Install beacon client ("+w.BeaconID+")", beaconClient),
		wireStep(),
		startStep(),
		handshakeStep(),
	}, nil
```

Add after `ownActiveUnitPorts` (i.e. after the preflight section):

```go
// ---------------------------------------------------------------------
// account
// ---------------------------------------------------------------------

// accountStep creates the dedicated unprivileged system account the node
// services run as (catalog.ServiceUser). The services drop to this user
// via the User=/Group= lines in the rendered units; setup itself still
// runs as root (it is what performs the useradd). Home is the valve-node
// data root purely as a bookkeeping convention — nothing reads it, so it
// is not created here (--no-create-home) and not chain-specific.
func accountStep() Step {
	return Step{
		ID:    "account",
		Title: "Create service account (" + catalog.ServiceUser + ")",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			cmd := fmt.Sprintf(
				"id -u %[1]s >/dev/null 2>&1 || useradd --system --user-group --home-dir /var/lib/valve-node --no-create-home --shell /usr/sbin/nologin %[1]s",
				catalog.ServiceUser,
			)
			res, err := e.Run(ctx, cmd, streamOpts(ctx, st, "account"))
			if err != nil {
				return fmt.Errorf("account: useradd: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("account: creating user %s failed (exit %d): %s",
					catalog.ServiceUser, res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			res, err := e.Run(ctx, "id -u "+catalog.ServiceUser, nil)
			if err != nil {
				return fmt.Errorf("account: id -u %s: %w", catalog.ServiceUser, err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("account: service user %s does not exist yet", catalog.ServiceUser)
			}
			return nil
		},
	}
}
```

Note: `catalog.ServiceUser` is a bare identifier in a shell command — it is a compile-time constant `valve-node` with no shell metacharacters, so `shQuote` is unnecessary (matches how unit names are interpolated today).

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/setup/ -v`
Expected: all PASS. Watch specifically for engine/steps tests that count or enumerate steps — `TestPlan_ReturnsOrderedSteps` is the known one; fix any other ordering assertion the same way (insert `"account"` after `"preflight"`).

- [ ] **Step 5: Commit**

```bash
git add internal/setup/steps.go internal/setup/steps_test.go
git commit -m "feat(setup): account step creates the valve-node service user"
```

---

### Task 3: Wire step owns the data tree; Verify enforces ownership (migration path)

**Files:**
- Modify: `internal/setup/steps.go` (`wireStep` Run at 301–377, Verify at 378–430)
- Test: `internal/setup/steps_test.go` (wire tests)

**Interfaces:**
- Consumes: `catalog.ServiceUser`/`ServiceGroup`; `jwtPathFor(w)` (steps.go:430–435); `shQuote`.
- Produces: wire Run always executes `mkdir -p <DataDir> && chown -R valve-node:valve-node <DataDir> <jwtPath>` after the JWT block and **before** writing/restarting units (services must never start against data they cannot write). Wire Verify additionally requires `stat -c %U <DataDir>` to report `valve-node`. Together with the existing content-diff restart, this makes "re-run setup" the complete migration path for root-era installs: units are rewritten with `User=`, data is re-owned, services restarted.

- [ ] **Step 1: Write the failing tests**

Append to `internal/setup/steps_test.go`:

```go
func TestWire_RunChownsDataDirToServiceUser(t *testing.T) {
	e := newFakeExecutor()
	step := wireStep()
	if err := step.Run(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	wantChown := fmt.Sprintf("mkdir -p '/mnt/reth' && chown -R %s:%s '/mnt/reth' '/mnt/reth/jwt.hex'",
		catalog.ServiceUser, catalog.ServiceGroup)
	var found bool
	var chownIdx, writeIdx int
	for i, c := range e.callLog() {
		if c == wantChown {
			found, chownIdx = true, i
		}
		if strings.HasPrefix(c, "WriteFile /etc/systemd/system/valve-node-exec.service") {
			writeIdx = i
		}
	}
	if !found {
		t.Fatalf("no chown command %q in calls: %v", wantChown, e.callLog())
	}
	if chownIdx > writeIdx {
		t.Fatalf("chown (call %d) must run before the units are written (call %d)", chownIdx, writeIdx)
	}
}

func TestWire_RunFailsWhenChownFails(t *testing.T) {
	e := newFakeExecutor().
		script("chown -R", executor.Result{ExitCode: 1, Stderr: "chown: changing ownership: read-only file system"})
	step := wireStep()
	err := step.Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("want error when chown fails, got nil")
	}
	if !strings.Contains(err.Error(), "read-only file system") {
		t.Fatalf("error %q does not surface chown stderr", err)
	}
}

func TestWire_VerifyFailsWhenDataDirNotOwnedByServiceUser(t *testing.T) {
	w := testWire()
	execUnit, beaconUnit, err := catalog.RenderUnits(w)
	if err != nil {
		t.Fatalf("RenderUnits: %v", err)
	}
	e := newFakeExecutor().
		script("test -f", executor.Result{ExitCode: 0}).
		script("systemctl is-enabled", executor.Result{Stdout: "enabled\nenabled\n", ExitCode: 0}).
		script("stat -c %U", executor.Result{Stdout: "root\n", ExitCode: 0})
	e.files["/etc/systemd/system/valve-node-exec.service"] = []byte(execUnit)
	e.files["/etc/systemd/system/valve-node-beacon.service"] = []byte(beaconUnit)
	step := wireStep()
	err = step.Verify(context.Background(), e, &State{Wire: w})
	if err == nil {
		t.Fatal("want Verify error while the data dir is still root-owned, got nil")
	}
	if !strings.Contains(err.Error(), "root") || !strings.Contains(err.Error(), catalog.ServiceUser) {
		t.Fatalf("error %q should name both the current owner and the wanted user", err)
	}

	e.scripts["stat -c %U"] = executor.Result{Stdout: catalog.ServiceUser + "\n", ExitCode: 0}
	if err := step.Verify(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("want Verify to pass once the data dir is owned by %s, got %v", catalog.ServiceUser, err)
	}
}
```

(`fmt` may need adding to the test file's imports.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/setup/ -run 'TestWire_RunChowns|TestWire_RunFailsWhenChown|TestWire_VerifyFailsWhenDataDirNotOwned' -v`
Expected: FAIL — no chown command found; Verify passes despite root owner.

- [ ] **Step 3: Implement the wire changes**

In `wireStep` Run (steps.go), directly after the JWT block (after line 322's closing brace) and before `catalog.RenderUnits(w)`:

```go
			// Re-own the whole data tree to the service user on every run.
			// This covers three cases with one command: the JWT just
			// written above as root, a fresh DataDir, and — the migration
			// path — a pre-de-root install whose entire tree is still
			// root-owned. It must happen before the units below are
			// (re)written and restarted, so the services never come up
			// unable to write their own data.
			chownCmd := fmt.Sprintf("mkdir -p %s && chown -R %s:%s %s %s",
				shQuote(w.DataDir), catalog.ServiceUser, catalog.ServiceGroup,
				shQuote(w.DataDir), shQuote(jwtPath))
			res, err := e.Run(ctx, chownCmd, opts)
			if err != nil {
				return fmt.Errorf("wire: chown data dir: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("wire: chown data dir failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
			}
```

Note the JWT probe result at line 306 is named `probe`, so `res, err :=` does not collide; if the compiler reports shadowing issues, rename locals rather than restructuring.

In `wireStep` Verify, after the files-exist check (after line 390's closing brace, before the `systemctl is-enabled` check):

```go
			// Ownership is part of "wired": a root-owned data dir means a
			// pre-de-root install that Run must migrate (chown -R).
			res, err = e.Run(ctx, fmt.Sprintf("stat -c %%U %s", shQuote(w.DataDir)), nil)
			if err != nil {
				return fmt.Errorf("wire: verify owner: %w", err)
			}
			if owner := strings.TrimSpace(res.Stdout); owner != catalog.ServiceUser {
				return fmt.Errorf("wire: %s is owned by %q, not the service user %q yet", w.DataDir, owner, catalog.ServiceUser)
			}
```

- [ ] **Step 4: Run the package tests and repair existing wire tests**

Run: `go test ./internal/setup/ -v`
Expected: the three new tests PASS. Existing wire tests that assert Verify passes will now FAIL because the unscripted `stat -c %U` returns empty stdout — fix each by scripting the fake:

```go
	.script("stat -c %U", executor.Result{Stdout: catalog.ServiceUser + "\n", ExitCode: 0})
```

Existing wire Run tests that enumerate the exact call sequence gain one `mkdir -p ... && chown -R ...` call after the JWT probe/write — insert it into their expected-call lists verbatim. Do not weaken any new assertion to make an old test pass.

- [ ] **Step 5: Run the whole suite**

Run: `go build ./... && go test ./...`
Expected: PASS everywhere. `internal/server` engine-flow tests use their own fakes — if any drive the full step list they need the same `stat -c %U` script and step-count updates; apply the identical fixes there.

- [ ] **Step 6: Commit**

```bash
git add internal/setup/steps.go internal/setup/steps_test.go
git commit -m "feat(setup): wire step re-owns the data tree to the service user; ownership is verified (migration path for root-era installs)"
```

---

### Task 4: Documentation — README requirements and v0.3 note

**Files:**
- Modify: `README.md` (lines 20–43 v0.2/Requirements sections, lines 91–97 "How it's built" root note)

**Interfaces:**
- Consumes: the shipped behavior from Tasks 1–3.
- Produces: user-facing docs stating services run unprivileged, setup still needs root, and that re-running setup migrates an existing install.

- [ ] **Step 1: Update the README**

Insert a `## v0.3 (unreleased)` section after the `## v0.2` paragraph (before `## Requirements`):

```markdown
## v0.3 (unreleased)

v0.3 de-roots the node services: the execution and beacon clients now run
as a dedicated unprivileged system user (`valve-node`) under hardened
systemd units (`NoNewPrivileges`, `ProtectSystem=strict` with the data
directory carved out, private `/tmp` and devices). Setup itself still
requires root — it creates the user, writes units, and owns the data
directory to the service account. Existing installs migrate automatically:
re-run setup against the target and the units are rewritten, the data
directory re-owned, and the services restarted.
```

Replace the last Requirements bullet (line 43):

```markdown
- Node services (the execution and beacon clients) run as the dedicated
  unprivileged `valve-node` system user, which setup creates. (In v0.1–v0.2
  they ran as root; re-running setup migrates an existing install.)
```

In "How it's built" (line 96), leave "Both modes need root on the target (see Requirements above)." unchanged — it is still true of *setup* — but extend the sentence: "Both modes need root on the target for setup itself (see Requirements above); the node services it installs run unprivileged."

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: v0.3 de-root — services run as unprivileged valve-node user"
```

---

## Self-Review

1. **Spec coverage:** dedicated unprivileged user (Task 2), hardened units (Task 1), migration path for existing root installs (Task 3 chown + existing content-diff restart; documented in Task 4). Setup-still-needs-root preserved (no change to preflight — deliberate). Privileged-port edge covered by conditional `AmbientCapabilities` since port validation allows 1–65535.
2. **Placeholder scan:** none — every step has full code, commands, and expected outcomes.
3. **Type consistency:** `catalog.ServiceUser`/`ServiceGroup` (Task 1) consumed by Tasks 2–3; `renderUnit(description, execStart string, w WireConfig, netBindCap bool)` matches both call sites; step ID `"account"` consistent between `Plan` and tests; `accountStep()`/`wireStep()` signatures match existing `Step` factories.

Known judgment calls locked in: single shared service user (both clients need to read the same JWT; per-service users would need a shared group and per-subtree perms for no practical gain at this stage); unconditional `chown -R` on every wire Run (idempotent, and an owner-guard would traverse the same inodes to prove a no-op); no `RestrictAddressFamilies`/`MemoryDenyWriteExecute` (breakage risk across four client implementations outweighs marginal hardening — revisit after soak).
