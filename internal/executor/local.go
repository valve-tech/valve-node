package executor

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// local runs commands and touches files on the machine it executes on.
type local struct{}

// NewLocal returns an Executor that runs commands on the local machine.
func NewLocal() Executor {
	return &local{}
}

func (l *local) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stderr = &stderrBuf

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return Result{}, err
	}

	if err := c.Start(); err != nil {
		return Result{}, err
	}

	scanErrCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutBuf.WriteString(line)
			stdoutBuf.WriteByte('\n')
			if opts != nil && opts.Stream != nil {
				opts.Stream(line)
			}
		}
		scanErrCh <- scanner.Err()
	}()

	scanErr := <-scanErrCh
	waitErr := c.Wait()

	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if scanErr != nil && scanErr != io.EOF {
		return Result{}, scanErr
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
	return os.WriteFile(path, content, mode)
}

func (l *local) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *local) Close() error {
	return nil
}
