package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
)

// testSSHD is an in-process sshd used to exercise the SSH Executor without
// touching a real remote host. Commands are executed locally via `sh -c` and
// exit codes are propagated, per the task brief.
type testSSHD struct {
	host    string
	port    int
	hostKey ssh.PublicKey
}

func startTestSSHD(t *testing.T) (testSSHD, string) {
	t.Helper()

	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}

	clientPub, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	clientKeyPath := writePrivateKey(t, clientPriv)
	_ = clientPub

	srv := &gliderssh.Server{
		PublicKeyHandler: func(ctx gliderssh.Context, key gliderssh.PublicKey) bool {
			return true
		},
		Handler: func(s gliderssh.Session) {
			c := exec.CommandContext(s.Context(), "sh", "-c", s.RawCommand())
			c.Stdout = s
			c.Stderr = s.Stderr()
			c.Stdin = s
			runErr := c.Run()
			code := 0
			if runErr != nil {
				if exitErr, ok := runErr.(*exec.ExitError); ok {
					code = exitErr.ExitCode()
				} else {
					code = 1
				}
			}
			_ = s.Exit(code)
		},
	}
	srv.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = srv.Close()
	})

	addr := ln.Addr().(*net.TCPAddr)
	return testSSHD{
		host:    "127.0.0.1",
		port:    addr.Port,
		hostKey: hostSigner.PublicKey(),
	}, clientKeyPath
}

func writePrivateKey(t *testing.T, priv ed25519.PrivateKey) string {
	t.Helper()

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	return path
}

func newSSHConfig(t *testing.T, d testSSHD, keyPath string) SSHConfig {
	t.Helper()
	return SSHConfig{
		Host:        d.host,
		User:        "testuser",
		KeyPath:     keyPath,
		HostKeyFile: filepath.Join(t.TempDir(), "known_hosts"),
		Port:        d.port,
	}
}

func TestSSH_Run_CapturesStdout(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
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

func TestSSH_Run_CapturesStderrSeparately(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
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

func TestSSH_Run_NonZeroExitIsNotAnError(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "exit 3", nil)
	if err != nil {
		t.Fatalf("Run returned error for non-zero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
}

func TestSSH_Run_StreamReceivesLinesInOrder(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
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

func TestSSH_WriteFile_ReadFile_RoundTrips(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
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

func TestSSH_TOFU_UnknownHostAppendsKey(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	cfg := newSSHConfig(t, d, keyPath)

	if _, err := os.Stat(cfg.HostKeyFile); err == nil {
		t.Fatalf("HostKeyFile unexpectedly exists before first connect")
	}

	e, err := NewSSH(cfg)
	if err != nil {
		t.Fatalf("NewSSH (first connect, unknown host): %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	info, err := os.Stat(cfg.HostKeyFile)
	if err != nil {
		t.Fatalf("HostKeyFile not created: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("HostKeyFile mode = %v, want 0600", info.Mode().Perm())
	}

	data, err := os.ReadFile(cfg.HostKeyFile)
	if err != nil {
		t.Fatalf("read HostKeyFile: %v", err)
	}
	addr := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	line := strings.TrimSpace(string(data))
	fields := strings.Fields(line)
	if len(fields) != 3 {
		t.Fatalf("HostKeyFile line = %q, want 3 whitespace-separated fields", line)
	}
	if fields[0] != addr {
		t.Errorf("HostKeyFile host field = %q, want %q", fields[0], addr)
	}
	if fields[1] != d.hostKey.Type() {
		t.Errorf("HostKeyFile keytype field = %q, want %q", fields[1], d.hostKey.Type())
	}
	gotKeyBytes, err := base64.StdEncoding.DecodeString(fields[2])
	if err != nil {
		t.Fatalf("decode base64 key field: %v", err)
	}
	if string(gotKeyBytes) != string(d.hostKey.Marshal()) {
		t.Errorf("HostKeyFile key bytes mismatch server host key")
	}

	// Second connect against the now-known host with the same key must
	// succeed without erroring and without duplicating the line.
	e2, err := NewSSH(cfg)
	if err != nil {
		t.Fatalf("NewSSH (second connect, known host): %v", err)
	}
	_ = e2.Close()

	data2, err := os.ReadFile(cfg.HostKeyFile)
	if err != nil {
		t.Fatalf("read HostKeyFile after second connect: %v", err)
	}
	if strings.Count(strings.TrimSpace(string(data2)), "\n")+1 != 1 {
		t.Errorf("HostKeyFile grew after known-host reconnect: %q", string(data2))
	}
}

func TestSSH_TOFU_MismatchedHostKeyErrors(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	cfg := newSSHConfig(t, d, keyPath)

	// Pre-populate the host key file with a DIFFERENT key for this host:port.
	_, wrongPub, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}
	wrongSigner, err := ssh.NewSignerFromKey(wrongPub)
	if err != nil {
		t.Fatalf("wrong signer: %v", err)
	}
	wrongKey := wrongSigner.PublicKey()

	addr := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	line := fmt.Sprintf("%s %s %s\n", addr, wrongKey.Type(), base64.StdEncoding.EncodeToString(wrongKey.Marshal()))
	if err := os.MkdirAll(filepath.Dir(cfg.HostKeyFile), 0o700); err != nil {
		t.Fatalf("mkdir HostKeyFile parent: %v", err)
	}
	if err := os.WriteFile(cfg.HostKeyFile, []byte(line), 0o600); err != nil {
		t.Fatalf("pre-write HostKeyFile: %v", err)
	}

	_, err = NewSSH(cfg)
	if err == nil {
		t.Fatalf("NewSSH: expected error for mismatched host key, got nil")
	}
	if !strings.Contains(err.Error(), "host key") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "host key")
	}
}
