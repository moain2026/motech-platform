//go:build !windows

package agent

import "os/exec"

// silentCmd on non-Windows is just a plain exec.Command — there are no console
// windows to hide. Keeps the agent buildable/testable on Linux/macOS.
func silentCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
