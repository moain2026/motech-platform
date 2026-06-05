//go:build !windows

package agent

// installScheduledTask is a no-op on non-Windows platforms.
func installScheduledTask(exePath string) error { return nil }

// uninstallScheduledTask is a no-op on non-Windows platforms.
func uninstallScheduledTask() error { return nil }

// stopRunningAgent is a no-op on non-Windows platforms.
func stopRunningAgent() {}
