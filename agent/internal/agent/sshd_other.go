//go:build !windows

package agent

// ensureSSHServer is a no-op on non-Windows platforms (dev/test).
func ensureSSHServer() error { return nil }
