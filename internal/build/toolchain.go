// Package build turns AhdCode source into a native executable by driving the
// existing frontend, lowering, and Go backend stages behind one API.
package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FindGoToolchain locates the Go toolchain used to build generated programs.
// PATH is authoritative; the well-known install locations are only a fallback.
func FindGoToolchain() (string, error) {
	if located, err := exec.LookPath("go"); err == nil {
		return located, nil
	}
	candidates := []string{}
	if root := os.Getenv("GOROOT"); root != "" {
		candidates = append(candidates, filepath.Join(root, "bin", "go"))
	}
	candidates = append(candidates,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/lib/go/bin/go",
	)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no Go toolchain was found on PATH or in the standard install locations")
}
