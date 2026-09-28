//go:build linux

package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// TestFileAtomicMoveRefusesARealCrossDeviceMove uses /dev/shm (a separate
// tmpfs on typical Linux hosts) to prove atomicMove fails instead of copying.
func TestFileAtomicMoveRefusesARealCrossDeviceMove(t *testing.T) {
	directory := t.TempDir()
	shm, err := os.MkdirTemp("/dev/shm", "ahd-exdev-")
	if err != nil {
		t.Skip("/dev/shm is not available")
	}
	defer os.RemoveAll(shm)
	var here, there syscall.Stat_t
	if syscall.Stat(directory, &here) != nil || syscall.Stat(shm, &there) != nil || here.Dev == there.Dev {
		t.Skip("/dev/shm is on the same filesystem as the test directory")
	}
	source := filepath.Join(shm, "file")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := FileAtomicMove(source, filepath.Join(directory, "moved")); err == nil || !strings.Contains(err.Error(), "never copies") {
		t.Fatalf("cross-device move: %v", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("a refused cross-device move lost its source")
	}
	if _, err := os.Lstat(filepath.Join(directory, "moved")); err == nil {
		t.Fatal("a refused cross-device move created its destination")
	}
}
