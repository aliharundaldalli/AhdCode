//go:build !darwin && !linux && !windows

package ahdruntime

import "errors"

func ahdDiskUsageOf(path string) (ahdDiskUsage, error) {
	return ahdDiskUsage{}, errors.New("disk inspection is not supported on this platform")
}
