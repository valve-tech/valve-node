//go:build unix

package executor

import (
	"os"
	"os/exec"
	"syscall"
)

// setupProcAttrs runs cmd in its own process group so that ctx cancellation
// can kill not just the direct `sh` PID but every descendant it spawned
// (including anything it backgrounded and detached from, e.g. `foo &`), by
// signaling the whole group. Without this, an orphaned child that inherited
// the stdout pipe's write end keeps that pipe open, and Run would otherwise
// hang reading it until the orphan exits on its own.
func setupProcAttrs(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		if c.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
}
