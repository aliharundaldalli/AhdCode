//go:build !windows

package main

import (
	"os"
	"syscall"
)

// processAlive reports whether this user can still see the recorded process.
// Signal 0 performs the permission and existence check without delivering
// anything.
func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// terminateProcess asks the recorded application to stop. The ordinary path
// is SIGTERM so the application can shut down; --force escalates to SIGKILL.
func terminateProcess(pid int, force bool) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if force {
		return process.Signal(syscall.SIGKILL)
	}
	return process.Signal(syscall.SIGTERM)
}
