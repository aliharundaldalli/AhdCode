//go:build unix

package ahdruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func ahdTemplateMissing(err error) bool {
	return os.IsNotExist(err)
}

func ahdTemplateComponentReparse(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

// ahdTemplateReadFile reads path without following a final symlink.
// O_NOFOLLOW applies to the last component. Ancestors are rejected by
// ahdTemplateEnsureContained. A parent replaced by a symlink after that
// check is the remaining race.
func ahdTemplateReadFile(path string) ([]byte, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	return ahdTemplateReadLimited(file, ahdTemplateMaxBytes)
}
