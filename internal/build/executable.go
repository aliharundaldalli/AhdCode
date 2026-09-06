package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// One policy governs every native executable this package produces, because a
// generated program is built at one path and launched from another, and the two
// must agree exactly.
//
// Windows is the reason the policy has to be explicit. `go build -o <path>`
// writes the path it is given verbatim, so an output name without an extension
// produces a valid PE file that os/exec then refuses to start: on Windows
// exec.Cmd.Start resolves the program through PATHEXT, and a name with no
// recognized extension yields exec.ErrNotFound, "executable file not found in
// %PATH%", whether or not the file exists. Windows also reports no execute
// permission bit for any regular file, so a Unix-style 0111 test can never
// succeed there.

// executableSuffixFor is the extension a native executable must carry on goos.
func executableSuffixFor(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}

// executablePathFor returns path carrying the executable suffix goos requires.
// A path that already ends in that suffix is returned unchanged, so applying
// the policy twice is harmless.
func executablePathFor(goos, path string) string {
	suffix := executableSuffixFor(goos)
	if suffix == "" || path == "" {
		return path
	}
	if strings.EqualFold(filepath.Ext(path), suffix) {
		return path
	}
	return path + suffix
}

// ExecutablePath returns path carrying this platform's executable suffix. Every
// caller that names a native executable — the run cache, the dev candidate, the
// `ahdcode build` output — goes through it, so the built artifact and the path
// handed to the launcher are always the same file.
func ExecutablePath(path string) string {
	return executablePathFor(runtime.GOOS, path)
}

// executableTempPatternFor turns a temporary-file prefix into an os.CreateTemp
// pattern that keeps the executable suffix last, where the operating system
// expects it: "partial-" becomes "partial-*.exe" on Windows.
func executableTempPatternFor(goos, prefix string) string {
	suffix := executableSuffixFor(goos)
	if suffix == "" {
		return prefix
	}
	return prefix + "*" + suffix
}

func executableTempPattern(prefix string) string {
	return executableTempPatternFor(runtime.GOOS, prefix)
}

// runnableModeFor reports whether a directory entry of this mode can be run as
// a native executable on goos. Windows carries no execute bit, so requiring one
// there would reject every file that has just been built.
func runnableModeFor(goos string, mode fs.FileMode) bool {
	if !mode.IsRegular() {
		return false
	}
	if goos == "windows" {
		return true
	}
	return mode&0o111 != 0
}

// runnableFile reports whether path is a regular file this platform can run.
func runnableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return runnableModeFor(runtime.GOOS, info.Mode())
}
