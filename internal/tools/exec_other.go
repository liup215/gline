//go:build !windows

package tools

import "os/exec"

// hideConsole is a no-op on non-Windows platforms; child processes never
// flash console windows there.
func hideConsole(cmd *exec.Cmd) {}
