//go:build windows

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const taskName = "MotechConnectAgent"

// taskXMLTemplate is a Task Scheduler definition that is far more resilient than
// the bare `schtasks` flags. It adds (per ops review):
//   - BootTrigger with a 1-min delay so the NetBird daemon socket is ready first
//   - RestartOnFailure: every 1 minute, up to 3 times
//   - StartWhenAvailable: catch up if the machine was off / trigger missed
//   - ExecutionTimeLimit PT0S: no time limit (long-running heartbeat loop)
//   - Runs as SYSTEM, HighestAvailable, hidden, allowed on battery
//
// %EXE% is replaced with the absolute stable exe path.
const taskXMLTemplate = `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.3" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Secure remote access agent (Motech / Al-Abbasi Soft).</Description>
    <URI>\MotechConnectAgent</URI>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <Delay>PT15S</Delay>
    </LogonTrigger>
    <BootTrigger>
      <Enabled>true</Enabled>
      <Delay>PT1M</Delay>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>%USERID%</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <Hidden>true</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>5</Priority>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>3</Count>
    </RestartOnFailure>
    <IdleSettings>
      <StopOnIdleEnd>false</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%EXE%</Command>
      <Arguments>run</Arguments>
    </Exec>
  </Actions>
</Task>`

// currentTaskUser returns the identity the scheduled task should run as.
// We run as the INTERACTIVE USER (not SYSTEM): on some hardened machines SYSTEM
// scheduled tasks are blocked by policy/AV, and the agent only needs user-level
// rights (it writes to administrators_authorized_keys via an elevated run level
// and talks to the NetBird daemon which is its own service). Falls back to the
// USERNAME/USERDOMAIN env vars.
func currentTaskUser() string {
	// Prefer the SID (whoami /user) over the account NAME. Names fail with
	// "No mapping between account names and security IDs was done" on localized
	// Windows or unusual domain formats; a SID always resolves. Task Scheduler
	// accepts a SID literal in <UserId>.
	if out, err := silentCmd("whoami", "/user", "/fo", "csv", "/nh").Output(); err == nil {
		// CSV: "domain\\user","S-1-5-21-..."
		fields := strings.Split(strings.TrimSpace(string(out)), ",")
		if len(fields) >= 2 {
			sid := strings.Trim(strings.TrimSpace(fields[len(fields)-1]), "\"")
			if strings.HasPrefix(sid, "S-1-") {
				return sid
			}
		}
	}
	// Fallback to the account name.
	if out, err := silentCmd("whoami").Output(); err == nil {
		if u := strings.TrimSpace(string(out)); u != "" {
			return u // e.g. "motech\\moain"
		}
	}
	dom := os.Getenv("USERDOMAIN")
	user := os.Getenv("USERNAME")
	if dom != "" && user != "" {
		return dom + "\\" + user
	}
	return user
}

// installScheduledTask registers a Scheduled Task (via XML) to run the agent at
// logon/boot as the INTERACTIVE USER. The XML approach lets us set
// RestartOnFailure, StartWhenAvailable and ExecutionTimeLimit which the bare
// flags cannot express.
func installScheduledTask(exePath string) error {
	if exePath == "" {
		return fmt.Errorf("empty exe path")
	}
	_ = silentCmd("schtasks", "/Delete", "/TN", taskName, "/F").Run()

	user := currentTaskUser()
	xml := strings.ReplaceAll(taskXMLTemplate, "%EXE%", exePath)
	xml = strings.ReplaceAll(xml, "%USERID%", user)

	// Write the XML as UTF-16LE with BOM (Task Scheduler expects this for /XML).
	tmp := filepath.Join(os.TempDir(), "motech-task.xml")
	if err := os.WriteFile(tmp, encodeUTF16LEWithBOM(xml), 0o644); err != nil {
		return fmt.Errorf("write task xml: %w", err)
	}
	defer os.Remove(tmp)

	if out, err := silentCmd("schtasks", "/Create", "/TN", taskName,
		"/XML", tmp, "/F").CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create /XML: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	// Kick it off immediately so we don't wait for the next boot.
	_ = silentCmd("schtasks", "/Run", "/TN", taskName).Run()
	return nil
}

// encodeUTF16LEWithBOM converts a UTF-8 string to UTF-16LE bytes with a leading
// BOM, which is the encoding schtasks /XML expects.
func encodeUTF16LEWithBOM(s string) []byte {
	runes := []rune(s)
	buf := make([]byte, 0, len(runes)*2+2)
	buf = append(buf, 0xFF, 0xFE) // UTF-16LE BOM
	for _, r := range runes {
		if r > 0xFFFF {
			r = 0xFFFD // replace astral chars (none expected in our template)
		}
		buf = append(buf, byte(r), byte(r>>8))
	}
	return buf
}

func uninstallScheduledTask() error {
	return silentCmd("schtasks", "/Delete", "/TN", taskName, "/F").Run()
}

// stopRunningAgent kills any running motech-connect.exe EXCEPT the current
// process, so the stable-location binary can be replaced without a file lock.
func stopRunningAgent() {
	self := os.Getpid()
	_ = silentCmd("taskkill", "/F", "/FI", fmt.Sprintf("PID ne %d", self),
		"/IM", "motech-connect.exe").Run()
	time.Sleep(1500 * time.Millisecond)
}
