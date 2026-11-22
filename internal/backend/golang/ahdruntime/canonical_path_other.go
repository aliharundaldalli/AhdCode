//go:build !windows

package ahdruntime

import "path/filepath"

func ahdCanonicalExistingPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
