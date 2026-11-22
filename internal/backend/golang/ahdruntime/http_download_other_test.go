//go:build !windows

package ahdruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func unwritableDownloadDirectory(t *testing.T, parent string) string {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root can write through POSIX directory mode bits")
	}
	path := filepath.Join(parent, "locked")
	if err := os.Mkdir(path, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o700) })
	requireDownloadDirectoryDeniesCreation(t, path)
	return path
}
