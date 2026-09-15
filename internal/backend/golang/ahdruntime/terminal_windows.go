//go:build windows

package ahdruntime

import (
	"syscall"
	"unsafe"
)

// ahdTerminalVirtualTerminalProcessing is ENABLE_VIRTUAL_TERMINAL_PROCESSING:
// the console interprets escape sequences instead of printing them.
const ahdTerminalVirtualTerminalProcessing = 0x0004

var ahdTerminalScreenBufferInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")

type ahdTerminalCoordinate struct{ x, y int16 }

type ahdTerminalRectangle struct{ left, top, right, bottom int16 }

// ahdTerminalBufferInfo is CONSOLE_SCREEN_BUFFER_INFO.
type ahdTerminalBufferInfo struct {
	size, cursor ahdTerminalCoordinate
	attributes   uint16
	window       ahdTerminalRectangle
	maximum      ahdTerminalCoordinate
}

// AhdTerminalIsTerminal reports whether fd is a console: reading the console
// mode succeeds only on one.
func AhdTerminalIsTerminal(fd uintptr) bool {
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(fd), &mode) == nil
}

// ahdTerminalSize reads the size of the console window, not of its scroll
// buffer.
func ahdTerminalSize(fd uintptr) (int, int, bool) {
	if ahdTerminalScreenBufferInfo.Find() != nil {
		return 0, 0, false
	}
	var info ahdTerminalBufferInfo
	result, _, _ := ahdTerminalScreenBufferInfo.Call(fd, uintptr(unsafe.Pointer(&info)))
	if result == 0 {
		return 0, 0, false
	}
	return int(info.window.right-info.window.left) + 1, int(info.window.bottom-info.window.top) + 1, true
}

// AhdTerminalSequences reports whether the console already has virtual
// terminal processing enabled. It reads the mode and never changes it.
func AhdTerminalSequences(fd uintptr) bool {
	var mode uint32
	if syscall.GetConsoleMode(syscall.Handle(fd), &mode) != nil {
		return false
	}
	return mode&ahdTerminalVirtualTerminalProcessing != 0
}
