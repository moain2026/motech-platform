package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// administratorsAuthorizedKeys is the Windows OpenSSH file that grants SSH
// access to members of the Administrators group.
const administratorsAuthorizedKeys = `C:\ProgramData\ssh\administrators_authorized_keys`

// installAuthorizedKey writes the public key into the OS authorized-keys file.
// On Windows it targets administrators_authorized_keys; elsewhere it logs the
// intended action (so the agent builds/tests cross-platform).
func installAuthorizedKey(line string) error {
	if runtime.GOOS != "windows" {
		// Non-Windows (dev/test): keep behavior safe and observable.
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(administratorsAuthorizedKeys), 0o755); err != nil {
		return err
	}
	// Merge: keep other admins' keys, replace only our previous Motech key.
	// We tag our line so rotation can swap just ours without wiping others.
	const tag = "motech-agent"
	newLine := strings.TrimSpace(line) + " " + tag + "\n"
	var kept []string
	if existing, err := os.ReadFile(administratorsAuthorizedKeys); err == nil {
		for _, l := range strings.Split(string(existing), "\n") {
			l = strings.TrimSpace(l)
			if l == "" || strings.Contains(l, tag) {
				continue // drop blanks and our old key
			}
			kept = append(kept, l)
		}
	}
	out := strings.Join(kept, "\n")
	if out != "" {
		out += "\n"
	}
	out += newLine
	if err := os.WriteFile(administratorsAuthorizedKeys, []byte(out), 0o600); err != nil {
		return err
	}
	return nil
}

// currentLoginUser returns the short OS login name (without domain) for
// connection-info purposes. Reported to the dashboard via heartbeat.
func currentLoginUser() string {
	// USERNAME on Windows, USER on Unix.
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return ""
}
