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

// compilerBinDirectory reports the directory holding this ahdcode executable,
// with symbolic links resolved. An installed release is reached through
// bin/ahdcode -> ../current/bin/ahdcode, so the unresolved path names a
// directory that holds neither the private toolchain nor the bundled helpers.
func compilerBinDirectory() (string, bool) {
	executable, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return filepath.Dir(executable), true
}

// FindGoToolchain locates the Go toolchain used to build generated programs.
// Installed releases prefer their own private Go toolchain. Source builds
// retain PATH and standard-location discovery for contributors.
func FindGoToolchain() (string, error) {
	if bin, ok := compilerBinDirectory(); ok {
		name := "go"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		candidate := filepath.Join(bin, "..", "libexec", "go", "bin", name)
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return filepath.Clean(candidate), nil
		}
	}
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
