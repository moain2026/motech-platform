//go:build windows

package agent

import (
	"os/exec"
	"syscall"
)

// hideWindow is the Windows process-creation flag that prevents a console
// window from flashing up for every child process (powershell, netsh,
// schtasks, netbird, icacls...). Without this, each command opens a black
// window during install — exactly what we must avoid.
const createNoWindow = 0x08000000 // CREATE_NO_WINDOW

// silentCmd builds an *exec.Cmd whose console window is fully hidden. Use this
// for EVERY external command the agent runs on Windows so the install/runtime
// is completely silent — only our own app/UI is ever visible.
func silentCmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return c
}
