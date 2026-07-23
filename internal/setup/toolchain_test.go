package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

func mustClient(t *testing.T, id string) catalog.Client {
	t.Helper()
	c, ok := catalog.ClientByID(id)
	if !ok {
		t.Fatalf("catalog missing %q", id)
	}
	return c
}

// ---- neededToolchains ----

func TestNeededToolchains_DedupsAndSorts(t *testing.T) {
	got := neededToolchains(mustClient(t, "go-pulse"), mustClient(t, "prysm-pulse"))
	want := []string{"go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("neededToolchains(go-pulse, prysm-pulse) = %v, want %v", got, want)
	}

	got = neededToolchains(mustClient(t, "reth"), mustClient(t, "lighthouse-pulse"))
	want = []string{"rust"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("neededToolchains(reth, lighthouse-pulse) = %v, want %v", got, want)
	}

	got = neededToolchains(mustClient(t, "geth"), mustClient(t, "lighthouse"))
	want = []string{"go", "rust"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("neededToolchains(geth, lighthouse) = %v, want %v", got, want)
	}
}

// ---- toolchain step: Run ----

// TestToolchain_GoPairInstallsGoNotRust locks in the go-pulse+prysm-pulse
// pair (both Toolchain=="go"): git already present, go missing — Run must
// apt-install golang-go and must never touch rustup.
func TestToolchain_GoPairInstallsGoNotRust(t *testing.T) {
	needed := neededToolchains(mustClient(t, "go-pulse"), mustClient(t, "prysm-pulse"))
	e := newFakeExecutor().
		script("command -v git", executor.Result{ExitCode: 0}).
		script("command -v apt-get", executor.Result{ExitCode: 0}).
		script("command -v go", executor.Result{ExitCode: 1})

	step := toolchainStep(needed)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var installedGo, installedRust, ranGitInstall bool
	for _, c := range e.callLog() {
		if strings.Contains(c, "golang-go") {
			installedGo = true
		}
		if strings.Contains(c, "rustup.rs") {
			installedRust = true
		}
		if strings.Contains(c, "apt-get install") && strings.Contains(c, " git ") {
			ranGitInstall = true
		}
	}
	if !installedGo {
		t.Fatalf("expected golang-go apt install; calls = %v", e.callLog())
	}
	if installedRust {
		t.Fatal("did not expect rustup install for an all-go pair")
	}
	if ranGitInstall {
		t.Fatal("git was already present — apt-get install for git should not have run")
	}
}

// TestToolchain_RustPairInstallsRustNotGo mirrors the above for the
// reth+lighthouse-pulse pair (both Toolchain=="rust"): cargo missing
// entirely — Run must curl-pipe rustup and must never touch golang-go.
func TestToolchain_RustPairInstallsRustNotGo(t *testing.T) {
	needed := neededToolchains(mustClient(t, "reth"), mustClient(t, "lighthouse-pulse"))
	e := newFakeExecutor().
		script("command -v git", executor.Result{ExitCode: 0}).
		script("command -v cargo", executor.Result{ExitCode: 1})

	step := toolchainStep(needed)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var installedRust, installedGo bool
	for _, c := range e.callLog() {
		if strings.Contains(c, "rustup.rs") {
			installedRust = true
		}
		if strings.Contains(c, "golang-go") {
			installedGo = true
		}
	}
	if !installedRust {
		t.Fatalf("expected rustup install; calls = %v", e.callLog())
	}
	if installedGo {
		t.Fatal("did not expect golang-go install for an all-rust pair")
	}
}

// TestToolchain_AlreadyPresentSkipsAptEntirely locks in the marker-consistent
// semantics: when git and the needed toolchain are already on the target,
// Run must never invoke apt-get (or rustup) at all.
func TestToolchain_AlreadyPresentSkipsAptEntirely(t *testing.T) {
	needed := neededToolchains(mustClient(t, "reth"), mustClient(t, "lighthouse-pulse"))
	e := newFakeExecutor().
		script("command -v git", executor.Result{ExitCode: 0}).
		script("command -v cargo", executor.Result{ExitCode: 0})

	step := toolchainStep(needed)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, c := range e.callLog() {
		if strings.Contains(c, "apt-get") {
			t.Fatalf("apt-get should never be invoked when git/cargo are already present; calls = %v", e.callLog())
		}
		if strings.Contains(c, "rustup.rs") {
			t.Fatalf("rustup should never be invoked when cargo is already present; calls = %v", e.callLog())
		}
	}
}

// TestToolchain_GitMissingInstallsViaApt locks in the always-ensure-git
// behavior independent of any client's Toolchain.
func TestToolchain_GitMissingInstallsViaApt(t *testing.T) {
	needed := neededToolchains(mustClient(t, "reth"), mustClient(t, "lighthouse-pulse"))
	e := newFakeExecutor().
		script("command -v git", executor.Result{ExitCode: 1}).
		script("command -v apt-get", executor.Result{ExitCode: 0}).
		script("command -v cargo", executor.Result{ExitCode: 0})

	step := toolchainStep(needed)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	if err := step.Run(context.Background(), e, st); err != nil {
		t.Fatalf("Run: %v", err)
	}

	ranGitInstall := false
	for _, c := range e.callLog() {
		if strings.Contains(c, "apt-get install") && strings.Contains(c, "git") {
			ranGitInstall = true
		}
	}
	if !ranGitInstall {
		t.Fatalf("expected apt-get install for git; calls = %v", e.callLog())
	}
}

// TestToolchain_FailsClearlyWhenAptMissing locks in the v1 target
// assumption: on a non-Debian/Ubuntu box (no apt-get) where git or go is
// missing, Run must fail with a clear "v1 supports Debian/Ubuntu targets"
// error rather than a raw shell "command not found".
func TestToolchain_FailsClearlyWhenAptMissing(t *testing.T) {
	needed := neededToolchains(mustClient(t, "go-pulse"), mustClient(t, "prysm-pulse"))
	e := newFakeExecutor().
		script("command -v git", executor.Result{ExitCode: 1}).
		script("command -v apt-get", executor.Result{ExitCode: 1})

	step := toolchainStep(needed)
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	err := step.Run(context.Background(), e, st)
	if err == nil {
		t.Fatal("want error when apt-get is missing on a non-Debian/Ubuntu target")
	}
	if !strings.Contains(err.Error(), "Debian/Ubuntu") {
		t.Fatalf("error should clearly say v1 supports Debian/Ubuntu targets, got: %v", err)
	}
}

// ---- toolchain step: Verify ----

func TestToolchain_VerifyPassesWhenGitAndGoAvailable(t *testing.T) {
	needed := neededToolchains(mustClient(t, "go-pulse"), mustClient(t, "prysm-pulse"))
	e := newFakeExecutor().
		script("git --version", executor.Result{ExitCode: 0}).
		script("go version", executor.Result{ExitCode: 0})

	step := toolchainStep(needed)
	if err := step.Verify(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestToolchain_VerifyFailsWhenCargoMissing(t *testing.T) {
	needed := neededToolchains(mustClient(t, "reth"), mustClient(t, "lighthouse-pulse"))
	e := newFakeExecutor().
		script("git --version", executor.Result{ExitCode: 0}).
		script("cargo --version", executor.Result{ExitCode: 127})

	step := toolchainStep(needed)
	if err := step.Verify(context.Background(), e, &State{Wire: testWire()}); err == nil {
		t.Fatal("want error when cargo --version fails, got nil")
	}
}

// TestToolchain_VerifySkipsRunWhenAlreadySatisfied exercises the full
// RunAll idempotence contract: a passing Verify pre-check must skip Run.
func TestToolchain_VerifySkipsRunWhenAlreadySatisfied(t *testing.T) {
	needed := neededToolchains(mustClient(t, "reth"), mustClient(t, "lighthouse-pulse"))
	e := newFakeExecutor().
		script("git --version", executor.Result{ExitCode: 0}).
		script("cargo --version", executor.Result{ExitCode: 0})

	step := toolchainStep(needed)
	runCalled := false
	steps := []Step{{
		ID:     step.ID,
		Verify: step.Verify,
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			runCalled = true
			return step.Run(ctx, e, st)
		},
	}}
	st := &State{Wire: testWire(), Events: make(chan Event, 100)}
	if err := RunAll(context.Background(), e, steps, st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if runCalled {
		t.Fatal("Run executed even though toolchain Verify pre-check already succeeded")
	}
}

// ---- Plan wiring ----

func TestPlan_InsertsToolchainStepSecond(t *testing.T) {
	steps, err := Plan(testWire())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(steps) < 2 || steps[1].ID != "toolchain" {
		var ids []string
		for _, s := range steps {
			ids = append(ids, s.ID)
		}
		t.Fatalf("Plan() step order = %v, want toolchain as step[1]", ids)
	}
}
