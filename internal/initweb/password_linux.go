//go:build linux

package initweb

import (
	"os"

	"golang.org/x/sys/unix"
)

func isTerminalFile(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	return err == nil
}

func disableEcho(fd int) (func(), error) {
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	previous := *state
	state.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, state); err != nil {
		return nil, err
	}
	return func() { _ = unix.IoctlSetTermios(fd, unix.TCSETS, &previous) }, nil
}
