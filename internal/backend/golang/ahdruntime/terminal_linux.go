//go:build linux

package ahdruntime

import (
	"syscall"
	"unsafe"
)

// ahdTerminalWindowSize is the kernel's struct winsize.
type ahdTerminalWindowSize struct {
	rows, columns, xPixels, yPixels uint16
}

func ahdTerminalIoctl(fd, request uintptr, argument unsafe.Pointer) bool {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(argument))
	return errno == 0
}

// AhdTerminalIsTerminal reports whether fd is a terminal: reading terminal
// attributes succeeds only on one.
func AhdTerminalIsTerminal(fd uintptr) bool {
	var attributes syscall.Termios
	return ahdTerminalIoctl(fd, syscall.TCGETS, unsafe.Pointer(&attributes))
}

// ahdTerminalSize reads the terminal's columns and rows.
func ahdTerminalSize(fd uintptr) (int, int, bool) {
	var size ahdTerminalWindowSize
	if !ahdTerminalIoctl(fd, syscall.TIOCGWINSZ, unsafe.Pointer(&size)) {
		return 0, 0, false
	}
	return int(size.columns), int(size.rows), true
}

// AhdTerminalSequences reports whether the terminal interprets escape
// sequences. Linux terminals do; TERM=dumb is handled by the color policy.
func AhdTerminalSequences(fd uintptr) bool { return true }
