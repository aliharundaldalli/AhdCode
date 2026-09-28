//go:build !windows

package ahdruntime

import (
	"os/exec"
	"syscall"
)

// ahdProcessConfigure starts the child in its own process group, so a
// timeout or output overflow kills the child and every descendant that stayed
// in that group. A descendant that deliberately starts its own session or
// group is outside this guarantee.
func ahdProcessConfigure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// A negative pid addresses the whole process group.
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
