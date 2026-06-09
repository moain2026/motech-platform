//go:build !windows

package agent

// ensureSSHServer is a no-op on non-Windows (dev/test): we don't manage the
// system sshd, ACLs, or firewall outside Windows.
func ensureSSHServer() error { return nil }

// VerifySSHReady on non-Windows reports a clear dev-mode message and no error,
// so the cross-platform install flow stays buildable/testable on Linux/macOS.
func VerifySSHReady(expectedPubKey string) (string, error) {
	return "بيئة تطوير (غير ويندوز): تخطّي فحص SSH", nil
}
