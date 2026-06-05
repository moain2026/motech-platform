//go:build windows

package agent

import (
	"os/exec"
)

// ensureSSHServer installs and starts the Windows OpenSSH Server, sets it to
// auto-start, fixes ACLs on administrators_authorized_keys, and opens the
// firewall for port 22. Idempotent — safe to run on every setup.
func ensureSSHServer() error {
	// 1. Install the OpenSSH.Server capability (no-op if already present).
	run("powershell", "-NoProfile", "-Command",
		`if (-not (Get-WindowsCapability -Online -Name OpenSSH.Server* | Where-Object State -eq 'Installed')) { Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0 }`)

	// 2. Start sshd and set it to start automatically.
	run("powershell", "-NoProfile", "-Command", `Set-Service -Name sshd -StartupType Automatic`)
	run("powershell", "-NoProfile", "-Command", `Start-Service sshd`)

	// 3. Open the firewall for SSH (port 22) if not already.
	run("powershell", "-NoProfile", "-Command",
		`if (-not (Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue)) { New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 }`)

	// 4. Fix ACL on administrators_authorized_keys (OpenSSH requires this:
	//    only Administrators + SYSTEM may have access, inheritance disabled).
	run("powershell", "-NoProfile", "-Command",
		`$p='C:\ProgramData\ssh\administrators_authorized_keys'; if (Test-Path $p) { icacls $p /inheritance:r /grant 'Administrators:F' /grant 'SYSTEM:F' | Out-Null }`)

	return nil
}

// run executes a command and ignores errors (best-effort, logged by caller).
func run(name string, args ...string) {
	_ = exec.Command(name, args...).Run()
}
