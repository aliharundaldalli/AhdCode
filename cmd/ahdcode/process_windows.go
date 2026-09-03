//go:build windows

package main

import "os"

// terminateOwnedProcess stops a process this CLI started and holds a handle
// to. There is deliberately no pid-based counterpart: nothing in AhdCode
// signals a process id that came out of a file. Windows has no graceful
// POSIX-style termination signal available here, so both the ordinary and the
// forced path use the platform's own terminate call.
func terminateOwnedProcess(process *os.Process, force bool) error {
	if process == nil {
		return os.ErrProcessDone
	}
	return process.Kill()
}
