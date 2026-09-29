//go:build linux

package ahdruntime

import "syscall"

// On Linux the block counts are in units of f_frsize when it is set.
func ahdDiskBlockSize(stat *syscall.Statfs_t) uint64 {
	if stat.Frsize > 0 {
		return uint64(stat.Frsize)
	}
	return uint64(stat.Bsize)
}
