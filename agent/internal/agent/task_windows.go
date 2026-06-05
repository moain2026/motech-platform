//go:build windows

package agent

import (
	"fmt"
	"os/exec"
)

// taskName is the Windows Scheduled Task name for the agent.
const taskName = "MotechConnectAgent"

// installScheduledTask registers a Scheduled Task that runs the agent's
// heartbeat loop as SYSTEM at every boot (and starts it now). This is more
// reliable than a Go Windows service for a long-running poll loop.
func installScheduledTask(exePath string) error {
	if exePath == "" {
		return fmt.Errorf("empty exe path")
	}
	// Remove any prior task (ignore error).
	_ = exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()

	// Create the task: run at startup, as SYSTEM, highest privileges.
	create := exec.Command("schtasks",
		"/Create",
		"/TN", taskName,
		"/TR", fmt.Sprintf(`"%s" run`, exePath),
		"/SC", "ONSTART",
		"/RU", "SYSTEM",
		"/RL", "HIGHEST",
		"/F",
	)
	if out, err := create.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create: %w (%s)", err, string(out))
	}

	// Start it right now too.
	if out, err := exec.Command("schtasks", "/Run", "/TN", taskName).CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks run: %w (%s)", err, string(out))
	}
	return nil
}

// uninstallScheduledTask removes the agent's Scheduled Task.
func uninstallScheduledTask() error {
	return exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()
}
