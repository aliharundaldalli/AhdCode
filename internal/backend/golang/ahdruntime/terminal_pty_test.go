//go:build darwin || linux

package ahdruntime

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func setTestTerminalSize(t *testing.T, terminal *os.File, columns, rows int) {
	t.Helper()
	size := ahdTerminalWindowSize{rows: uint16(rows), columns: uint16(columns)}
	if !ahdTerminalIoctl(terminal.Fd(), syscall.TIOCSWINSZ, unsafe.Pointer(&size)) {
		t.Fatalf("could not set the pseudo-terminal size to %dx%d", columns, rows)
	}
}

// The terminal queries are checked against a real pseudo-terminal opened
// through the kernel, not a stub, and the color policy against the process
// environment exactly as a program sees it.
func TestTerminalQueriesOnARealPseudoTerminal(t *testing.T) {
	_, terminal := openTestTerminal(t)
	fd := terminal.Fd()
	if !AhdTerminalIsTerminal(fd) {
		t.Fatal("the pseudo-terminal is not reported as a terminal")
	}
	setTestTerminalSize(t, terminal, 132, 43)
	if width, height := AhdTerminalDimensions(fd); width == nil || *width != 132 || height == nil || *height != 43 {
		t.Fatalf("dimensions = %v x %v, want 132 x 43", width, height)
	}
	setTestTerminalSize(t, terminal, 0, 0)
	if width, height := AhdTerminalDimensions(fd); width != nil || height != nil {
		t.Fatalf("a zero-sized terminal has dimensions %v x %v, want null", width, height)
	}
	setTestTerminalSize(t, terminal, 80, 0)
	if width, height := AhdTerminalDimensions(fd); width == nil || *width != 80 || height != nil {
		t.Fatalf("dimensions = %v x %v, want 80 x null", width, height)
	}
	if !AhdTerminalSequences(fd) {
		t.Fatal("a Unix terminal does not report escape sequence support")
	}

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")
	if !AhdTerminalColorEnabled(fd) {
		t.Fatal("an interactive terminal with an empty NO_COLOR does not support color")
	}
	styled, _ := AhdTerminalStyle("OK", "green", "default", false, false, AhdTerminalColorEnabled(fd))
	if styled != "\x1b[32mOK\x1b[0m" {
		t.Fatalf("styled text on a color terminal = %q", styled)
	}
	t.Setenv("TERM", "dumb")
	if AhdTerminalColorEnabled(fd) {
		t.Fatal("TERM=dumb still supports color")
	}
	t.Setenv("TERM", "xterm")
	t.Setenv("NO_COLOR", "1")
	if AhdTerminalColorEnabled(fd) {
		t.Fatal("NO_COLOR=1 still supports color")
	}
}
