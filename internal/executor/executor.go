// Package executor abstracts running shell commands and reading/writing
// files either on the local machine or on a remote host over SSH. It is the
// architectural seam that every later setup step, monitor, and log tailer in
// valve-node runs through.
package executor

import (
	"context"
	"io/fs"
)

// Result is the outcome of a Run call. A non-zero ExitCode is NOT reported as
// a Go error — error is reserved for transport/spawn/context failures.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// StreamFunc is called once per stdout line, in order, as it arrives.
type StreamFunc func(line string)

// RunOpts controls Run behavior. Stream may be nil.
type RunOpts struct {
	Stream StreamFunc
}

// Executor runs commands and moves files, either locally or over SSH.
type Executor interface {
	// Run executes cmd via `sh -c`. opts may be nil. If opts.Stream is set,
	// it receives stdout lines live, in order, as the command runs.
	Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error)
	WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error
	ReadFile(ctx context.Context, path string) ([]byte, error)
	Close() error
}

// SSHConfig configures NewSSH.
type SSHConfig struct {
	Host        string
	User        string
	KeyPath     string
	HostKeyFile string
	Port        int // 0 => 22
}
