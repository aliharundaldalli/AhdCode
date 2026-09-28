//go:build windows

package ahdruntime

import "os/exec"

// ahdProcessConfigure kills only the direct child on timeout or output
// overflow. Windows has no process groups in the Unix sense; descendants the
// child started are not guaranteed to be terminated.
func ahdProcessConfigure(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Kill()
	}
}
