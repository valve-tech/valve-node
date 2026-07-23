//go:build windows

package executor

import "os/exec"

// setupProcAttrs is a no-op on windows: local mode is a controller-only
// convenience (node targets are Linux), so we don't bother with process-group
// semantics here. c.Cancel falls back to CommandContext's default kill
// behavior (Process.Kill on the direct child only).
func setupProcAttrs(c *exec.Cmd) {}
