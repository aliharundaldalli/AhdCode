//go:build !darwin && !linux && !windows

package ahdruntime

// On a platform AhdCode does not target, no output is treated as a terminal:
// Terminal.isInteractive is false, the dimensions are null, and styled text
// degrades to plain text.

// AhdTerminalIsTerminal reports that fd is not a known terminal.
func AhdTerminalIsTerminal(fd uintptr) bool { return false }

func ahdTerminalSize(fd uintptr) (int, int, bool) { return 0, 0, false }

// AhdTerminalSequences reports that escape sequences are not known to work.
func AhdTerminalSequences(fd uintptr) bool { return false }
