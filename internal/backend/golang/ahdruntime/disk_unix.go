//go:build darwin || linux

package ahdruntime

import (
	"errors"
	"syscall"
)

// ahdDiskUsageOf reads statfs(2) for the filesystem containing path. Block
// counts are in the filesystem's fundamental block size (f_frsize on Linux,
// f_bsize on macOS). free counts every free block, including any reserved for
// the superuser; available counts the blocks an unprivileged process may use.
func ahdDiskUsageOf(path string) (ahdDiskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return ahdDiskUsage{}, errors.New("the filesystem information is unavailable: " + ahdFSReason(err))
	}
	size := ahdDiskBlockSize(&stat)
	total, ok1 := ahdDiskBytes(uint64(stat.Blocks), size)
	free, ok2 := ahdDiskBytes(uint64(stat.Bfree), size)
	available, ok3 := ahdDiskBytes(uint64(stat.Bavail), size)
	if !ok1 || !ok2 || !ok3 {
		return ahdDiskUsage{}, errors.New("the filesystem is too large to represent in bytes")
	}
	return ahdDiskUsage{Total: total, Free: free, Available: available}, nil
}
