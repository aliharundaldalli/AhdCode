//go:build darwin

package ahdruntime

import (
	"bytes"
	"os"
	"syscall"
	"testing"
	"unsafe"
)

// openTestTerminal opens a pseudo-terminal pair and returns its controlling
// and terminal sides. A machine without pseudo-terminals skips the test.
func openTestTerminal(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	control, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal is available: %v", err)
	}
	t.Cleanup(func() { control.Close() })
	fd := control.Fd()
	if !ahdTerminalIoctl(fd, syscall.TIOCPTYGRANT, nil) || !ahdTerminalIoctl(fd, syscall.TIOCPTYUNLK, nil) {
		t.Skip("the pseudo-terminal could not be unlocked")
	}
	var name [128]byte
	if !ahdTerminalIoctl(fd, syscall.TIOCPTYGNAME, unsafe.Pointer(&name[0])) {
		t.Skip("the pseudo-terminal name could not be read")
	}
	length := bytes.IndexByte(name[:], 0)
	if length < 0 {
		length = len(name)
	}
	terminal, err := os.OpenFile(string(name[:length]), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("the pseudo-terminal could not be opened: %v", err)
	}
	t.Cleanup(func() { terminal.Close() })
	return control, terminal
}
