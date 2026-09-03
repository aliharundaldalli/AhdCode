//go:build !windows

package main

import (
	"os"
	"syscall"
)

// terminateOwnedProcess stops a process this CLI started and holds a handle
// to. There is deliberately no pid-based counterpart: nothing in AhdCode
// signals a process id that came out of a file. The ordinary path is SIGTERM
// so the application can shut down; force escalates to SIGKILL.
func terminateOwnedProcess(process *os.Process, force bool) error {
	if process == nil {
		return os.ErrProcessDone
	}
	if force {
		return process.Signal(syscall.SIGKILL)
	}
	return process.Signal(syscall.SIGTERM)
}
