//go:build !windows

package agent

// acquireSingleInstance is a no-op on non-Windows platforms.
func acquireSingleInstance() (release func(), ok bool) {
	return func() {}, true
}
