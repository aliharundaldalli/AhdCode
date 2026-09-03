//go:build windows

package main

import "os"

// processAlive reports whether the recorded process still exists. On Windows
// os.FindProcess fails outright for a process that is gone.
func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = process.Release()
	return true
}

// terminateProcess stops the recorded application. Windows has no graceful
// POSIX-style termination signal available here, so both the ordinary and the
// forced path use the platform's own terminate call.
func terminateProcess(pid int, force bool) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
