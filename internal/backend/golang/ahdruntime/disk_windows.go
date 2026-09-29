//go:build windows

package ahdruntime

import (
	"errors"
	"syscall"
	"unsafe"
)

var ahdDiskFreeSpace = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// ahdDiskUsageOf asks GetDiskFreeSpaceExW about the volume containing path.
// total and free describe the whole volume; available is what this user may
// use (it reflects per-user quotas).
func ahdDiskUsageOf(path string) (ahdDiskUsage, error) {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ahdDiskUsage{}, errors.New("the path is not valid")
	}
	var available, total, free uint64
	result, _, callErr := ahdDiskFreeSpace.Call(uintptr(unsafe.Pointer(pointer)),
		uintptr(unsafe.Pointer(&available)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
	if result == 0 {
		return ahdDiskUsage{}, errors.New("the filesystem information is unavailable: " + ahdFSReason(callErr))
	}
	return ahdDiskUsage{Total: total, Free: free, Available: available}, nil
}
