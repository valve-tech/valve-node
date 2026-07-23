package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocal_Run_CapturesStdout(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "echo hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestLocal_Run_CapturesStderrSeparately(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "echo out; echo err 1>&2", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "out\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "out\n")
	}
	if res.Stderr != "err\n" {
		t.Errorf("Stderr = %q, want %q", res.Stderr, "err\n")
	}
}

func TestLocal_Run_NonZeroExitIsNotAnError(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "exit 3", nil)
	if err != nil {
		t.Fatalf("Run returned error for non-zero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
}

func TestLocal_Run_StreamReceivesLinesInOrder(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	var lines []string
	opts := &RunOpts{Stream: func(line string) {
		lines = append(lines, line)
	}}

	res, err := e.Run(context.Background(), `printf 'a\nb\n'`, opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("streamed lines = %v, want [a b]", lines)
	}
	if res.Stdout != "a\nb\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "a\nb\n")
	}
}

func TestLocal_WriteFile_ReadFile_RoundTrips(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")
	content := []byte("hello world\n")

	if err := e.WriteFile(context.Background(), path, content, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := e.ReadFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadFile content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o640))
	}
}

func TestLocal_Run_CtxCancelKillsFast(t *testing.T) {
	e := NewLocal()
	t.Cleanup(func() { _ = e.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := e.Run(ctx, "sleep 5", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from ctx cancel, got nil")
	}
	if elapsed >= time.Second {
		t.Errorf("Run took %v, want < 1s", elapsed)
	}
}
