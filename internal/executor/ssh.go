package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshExecutor runs commands and moves files on a remote host over SSH.
type sshExecutor struct {
	client *ssh.Client
}

// NewSSH dials user@host:port (default port 22) using the private key at
// cfg.KeyPath, verifying the remote host key against cfg.HostKeyFile using a
// trust-on-first-use policy: an unknown host's key is appended to
// HostKeyFile (created 0600 on first use); a known host presenting a
// different key is rejected with an error.
func NewSSH(cfg SSHConfig) (Executor, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	keyBytes, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", cfg.KeyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", cfg.KeyPath, err)
	}

	config := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: tofuHostKeyCallback(cfg.HostKeyFile),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return &sshExecutor{client: client}, nil
}

// tofuHostKeyCallback implements trust-on-first-use host key verification
// backed by a flat file of "host:port keytype base64key" lines.
func tofuHostKeyCallback(hostKeyFile string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		known, err := lookupHostKey(hostKeyFile, hostname)
		if err != nil {
			return err
		}
		if known == nil {
			return appendHostKey(hostKeyFile, hostname, key)
		}
		if !bytes.Equal(known.Marshal(), key.Marshal()) {
			return fmt.Errorf("host key mismatch for %s: presented %s key does not match the key on record in %s (possible man-in-the-middle attack, or the host was rebuilt)", hostname, key.Type(), hostKeyFile)
		}
		return nil
	}
}

// lookupHostKey returns the recorded public key for hostname in hostKeyFile,
// or nil if hostKeyFile doesn't exist or has no entry for hostname.
func lookupHostKey(hostKeyFile, hostname string) (ssh.PublicKey, error) {
	data, err := os.ReadFile(hostKeyFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read host key file %s: %w", hostKeyFile, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != hostname {
			continue
		}
		keyBytes, err := base64.StdEncoding.DecodeString(fields[2])
		if err != nil {
			return nil, fmt.Errorf("host key file %s: malformed entry for %s: %w", hostKeyFile, hostname, err)
		}
		key, err := ssh.ParsePublicKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("host key file %s: malformed entry for %s: %w", hostKeyFile, hostname, err)
		}
		return key, nil
	}
	return nil, nil
}

// appendHostKey records key for hostname in hostKeyFile, creating the file
// with mode 0600 if it doesn't already exist.
func appendHostKey(hostKeyFile, hostname string, key ssh.PublicKey) error {
	if dir := filepath.Dir(hostKeyFile); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create host key file dir: %w", err)
		}
	}

	f, err := os.OpenFile(hostKeyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open host key file %s: %w", hostKeyFile, err)
	}
	defer f.Close()

	line := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write host key file %s: %w", hostKeyFile, err)
	}
	return nil
}

func (s *sshExecutor) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("new ssh session: %w", err)
	}
	defer session.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = session.Close()
		case <-done:
		}
	}()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stderr = &stderrBuf

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("ssh stdout pipe: %w", err)
	}

	if err := session.Start(cmd); err != nil {
		return Result{}, fmt.Errorf("start ssh command: %w", err)
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
	waitErr := session.Wait()

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
		var exitErr *ssh.ExitError
		if errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitStatus()
			return result, nil
		}
		return Result{}, waitErr
	}

	return result, nil
}

// WriteFile writes content to path on the remote host by piping a
// base64-encoded copy through the remote `base64 -d` and setting mode with
// `chmod`. No SFTP subsystem is required.
func (s *sshExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	encoded := base64.StdEncoding.EncodeToString(content)
	cmd := fmt.Sprintf(
		"mkdir -p %s && printf %%s %s | base64 -d > %s && chmod %o %s",
		shQuote(filepath.Dir(path)),
		shQuote(encoded),
		shQuote(path),
		mode.Perm(),
		shQuote(path),
	)
	res, err := s.Run(ctx, cmd, nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("write remote file %s: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return nil
}

// ReadFile reads path from the remote host via `base64 < path` over Run.
func (s *sshExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) {
	cmd := fmt.Sprintf("base64 < %s", shQuote(path))
	res, err := s.Run(ctx, cmd, nil)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read remote file %s: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(res.Stdout), "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decode remote file %s: %w", path, err)
	}
	return decoded, nil
}

func (s *sshExecutor) Close() error {
	return s.client.Close()
}

// shQuote wraps s in single quotes for safe embedding in a `sh -c` command,
// escaping any embedded single quotes.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
