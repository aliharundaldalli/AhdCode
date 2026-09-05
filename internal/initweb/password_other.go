//go:build !darwin && !linux

package initweb

import "os"

func isTerminalFile(file *os.File) bool { return false }

func disableEcho(fd int) (func(), error) {
	return func() {}, nil
}
