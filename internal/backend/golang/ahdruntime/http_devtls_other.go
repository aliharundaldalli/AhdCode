//go:build !windows && !unix

package ahdruntime

import (
	"fmt"
	"os"
	"path/filepath"
)

func ahdDevTLSMissing(err error) bool {
	return os.IsNotExist(err)
}

func ahdDevTLSComponentReparse(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

func ahdDevTLSReadCredential(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is a symbolic link or non-regular file", filepath.Base(path))
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ahdDevTLSReadLimited(file)
}
