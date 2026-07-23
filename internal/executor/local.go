package executor

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// local runs commands and touches files on the machine it executes on.
type local struct{}

// NewLocal returns an Executor that runs commands on the local machine.
func NewLocal() Executor {
	return &local{}
}

func (l *local) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)

	// See proc_unix.go / proc_windows.go: on unix this runs cmd in its own
	// process group so ctx cancellation can kill every descendant it spawned,
	// not just the direct `sh` PID.
	setupProcAttrs(c)
	// Backstop: if killing the process group somehow doesn't unblock our
	// stdout read within this window (e.g. a doubly-detached daemon in a
	// different process group), exec forcibly closes the stdout pipe so Run
	// can still return instead of hanging forever.
	c.WaitDelay = 2 * time.Second

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stderr = &stderrBuf

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return Result{}, err
	}

	if err := c.Start(); err != nil {
		return Result{}, err
	}

	var streamFn StreamFunc
	if opts != nil {
		streamFn = opts.Stream
	}
	w := &lineStreamer{buf: &stdoutBuf, fn: streamFn}

	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, stdoutPipe)
		copyErrCh <- err
	}()

	// It is incorrect to call Wait before all reads from the StdoutPipe have
	// completed, so wait for the copy goroutine first.
	copyErr := <-copyErrCh
	w.Flush()
	waitErr := c.Wait()

	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if copyErr != nil {
		return Result{}, copyErr
	}

	result := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if waitErr != nil {
		exitErr, ok := waitErr.(*exec.ExitError)
		if !ok {
			return Result{}, waitErr
		}
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, nil
}

func (l *local) WriteFile(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	// os.WriteFile only applies mode on create; an existing file keeps its
	// old permissions, and umask can still narrow them even then. Chmod
	// unconditionally so mode is authoritative in both cases.
	return os.Chmod(path, mode)
}

func (l *local) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *local) Close() error {
	return nil
}
