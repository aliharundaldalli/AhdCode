//go:build darwin

package ahdruntime

import "syscall"

func ahdDiskBlockSize(stat *syscall.Statfs_t) uint64 { return uint64(stat.Bsize) }
