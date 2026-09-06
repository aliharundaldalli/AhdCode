package evaluator

import (
	"os"
	"path/filepath"
)

// interpreterBinDirectory reports the directory holding this ahdcode
// executable, with symbolic links resolved. An installed release is reached
// through bin/ahdcode -> ../current/bin/ahdcode, so the unresolved path names
// a directory that holds none of the bundled helpers.
func interpreterBinDirectory() (string, bool) {
	executable, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return filepath.Dir(executable), true
}
