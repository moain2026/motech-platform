//go:build windows

package agent

import (
	"syscall"
	"unsafe"
)

// acquireSingleInstance creates a named global mutex so only ONE motech-connect
// run-loop can be active machine-wide. If the mutex already exists, this process
// is a duplicate and ok=false is returned (caller should exit cleanly).
//
// We saw "loop start" logged twice in earlier tests (a SYSTEM task process and a
// leftover user process running simultaneously). A Global\ mutex prevents that
// regardless of session, so heartbeats never double-fire.
func acquireSingleInstance() (release func(), ok bool) {
	const errAlreadyExists = 183 // ERROR_ALREADY_EXISTS

	name, err := syscall.UTF16PtrFromString(`Global\MotechConnectAgent`)
	if err != nil {
		return func() {}, true // fail-open: don't block the agent on a name error
	}

	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex := modkernel32.NewProc("CreateMutexW")
	procCloseHandle := modkernel32.NewProc("CloseHandle")

	handle, _, lastErr := procCreateMutex.Call(
		0, // default security attributes
		0, // not initially owned
		uintptr(unsafe.Pointer(name)),
	)
	if handle == 0 {
		return func() {}, true // fail-open
	}

	if errno, isErrno := lastErr.(syscall.Errno); isErrno && int(errno) == errAlreadyExists {
		// Another instance already holds the mutex. Close our handle and bail.
		procCloseHandle.Call(handle)
		return func() {}, false
	}

	release = func() { procCloseHandle.Call(handle) }
	return release, true
}
