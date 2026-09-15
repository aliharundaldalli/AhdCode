//go:build linux

package ahdruntime

import (
	"os"
	"strconv"
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
	var unlock int32
	if !ahdTerminalIoctl(fd, syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)) {
		t.Skip("the pseudo-terminal could not be unlocked")
	}
	var number uint32
	if !ahdTerminalIoctl(fd, syscall.TIOCGPTN, unsafe.Pointer(&number)) {
		t.Skip("the pseudo-terminal number could not be read")
	}
	terminal, err := os.OpenFile("/dev/pts/"+strconv.FormatUint(uint64(number), 10), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("the pseudo-terminal could not be opened: %v", err)
	}
	t.Cleanup(func() { terminal.Close() })
	return control, terminal
}
