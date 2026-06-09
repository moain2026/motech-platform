//go:build windows

package agent

import (
	"fmt"
	"os"
	"strings"
)

// ensureSSHServer installs and starts the Windows OpenSSH Server, sets it to
// auto-start, fixes ACLs on administrators_authorized_keys, and opens the
// firewall for port 22. Idempotent — safe to run on every setup.
//
// Every command runs through silentCmd so NO console window ever appears.
func ensureSSHServer() error {
	// 1. Install the OpenSSH.Server capability (no-op if already present).
	run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
		`if (-not (Get-WindowsCapability -Online -Name OpenSSH.Server* | Where-Object State -eq 'Installed')) { Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0 }`)

	// 2. Start sshd and set it to start automatically.
	run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", `Set-Service -Name sshd -StartupType Automatic`)
	run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", `Start-Service sshd`)

	// 3. Open the firewall for SSH (port 22) if not already.
	run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
		`if (-not (Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue)) { New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 }`)

	// 4. Fix ACL on administrators_authorized_keys (OpenSSH requires this:
	//    only Administrators + SYSTEM may have access, inheritance disabled).
	run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
		`$p='C:\ProgramData\ssh\administrators_authorized_keys'; if (Test-Path $p) { icacls $p /inheritance:r /grant 'Administrators:F' /grant 'SYSTEM:F' | Out-Null }`)

	return nil
}

// VerifySSHReady confirms the SSH layer is actually ready after setup:
//  1. sshd service is Running.
//  2. our public key is present in administrators_authorized_keys.
//
// Returns a human-readable status and an error if anything is not ready, so the
// installer can report a precise "ready / not ready" instead of guessing.
func VerifySSHReady(expectedPubKey string) (string, error) {
	// 1. sshd running?
	out, _ := run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
		`(Get-Service sshd -ErrorAction SilentlyContinue).Status`)
	status := strings.TrimSpace(out)
	if !strings.EqualFold(status, "Running") {
		return "sshd ليست قيد التشغيل (الحالة: " + status + ")", fmt.Errorf("sshd not running: %q", status)
	}

	// 2. our key installed?
	const akf = `C:\ProgramData\ssh\administrators_authorized_keys`
	data, err := os.ReadFile(akf)
	if err != nil {
		return "ملف authorized_keys غير موجود", fmt.Errorf("read authorized_keys: %w", err)
	}
	// Compare on the base64 body of the key (ignore comment/tag differences).
	want := keyBody(expectedPubKey)
	if want == "" || !strings.Contains(string(data), want) {
		return "مفتاح SSH لم يُرفع في authorized_keys", fmt.Errorf("public key not found in authorized_keys")
	}

	return "sshd تعمل ومفتاح SSH مرفوع بنجاح", nil
}

// keyBody extracts the base64 payload of an SSH public key line (the part that
// uniquely identifies the key), ignoring the type prefix and trailing comment.
func keyBody(pub string) string {
	fields := strings.Fields(strings.TrimSpace(pub))
	if len(fields) >= 2 {
		return fields[1]
	}
	return ""
}

// run executes a command fully hidden (no console window) and returns its
// combined output. Best-effort: callers decide whether to act on the error.
func run(name string, args ...string) (string, error) {
	out, err := silentCmd(name, args...).CombinedOutput()
	return string(out), err
}
