//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const taskName = "MotechConnectAgent"

// installScheduledTask registers a Scheduled Task to run the agent at startup.
// Now that the agent no longer writes to os.Stdout (which crashed it under
// Session 0), running as SYSTEM ONSTART is viable. We add a 1-min delay so the
// NetBird daemon socket is ready first (avoids the boot race).
func installScheduledTask(exePath string) error {
	if exePath == "" {
		return fmt.Errorf("empty exe path")
	}
	_ = exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()
	args := []string{
		"/Create", "/TN", taskName,
		"/TR", fmt.Sprintf(`"%s" run`, exePath),
		"/SC", "ONSTART",
		"/DELAY", "0001:00", // 1 minute after boot
		"/RU", "SYSTEM",
		"/RL", "HIGHEST",
		"/F",
	}
	if out, err := exec.Command("schtasks", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	_ = exec.Command("schtasks", "/Run", "/TN", taskName).Run()
	return nil
}

func uninstallScheduledTask() error {
	return exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()
}

// stopRunningAgent kills any running motech-connect.exe EXCEPT the current
// process, so the stable-location binary can be replaced without a file lock.
func stopRunningAgent() {
	self := os.Getpid()
	_ = exec.Command("taskkill", "/F", "/FI", fmt.Sprintf("PID ne %d", self),
		"/IM", "motech-connect.exe").Run()
	time.Sleep(1500 * time.Millisecond)
}
