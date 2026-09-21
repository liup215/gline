//go:build windows

package tools

import (
	"os/exec"
	"syscall"
)

// hideConsole prevents child processes from flashing a console window when
// gline runs as a desktop GUI app on Windows. The flags are harmless inside
// a terminal session.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
