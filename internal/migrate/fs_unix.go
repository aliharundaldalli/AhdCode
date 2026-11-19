//go:build !windows

package migrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func isMissing(err error) bool {
	return os.IsNotExist(err)
}

func componentReparse(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

// openLockFile creates or opens the lock file without following a symlink.
// A non-regular leaf is rejected. A parent directory replaced after
// ensurePathContained is still the remaining race.
func openLockFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	reparse, reparseErr := componentReparse(path)
	if reparseErr != nil || reparse || !info.Mode().IsRegular() {
		_ = file.Close()
		if reparseErr != nil {
			return nil, reparseErr
		}
		return nil, fmt.Errorf("%s is a symbolic link or non-regular file; migrate will not use it as a lock", filepath.Base(path))
	}
	return file, nil
}

func lockOS(file *os.File) error {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("could not lock the migration directory: %s", err.Error())
	}
	return nil
}

func unlockOS(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

// readFileNoFollow reads path without following a final symlink.
// O_NOFOLLOW applies to the last component. Ancestors are rejected by
// ensurePathContained. A parent replaced by a symlink after that check is
// the remaining race.
func readFileNoFollow(path string) ([]byte, error) {
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
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	return io.ReadAll(file)
}
