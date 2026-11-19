//go:build windows

package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func isMissing(err error) bool {
	return os.IsNotExist(err) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND)
}

func componentReparse(path string) (bool, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attributes, err := windows.GetFileAttributes(pointer)
	if err != nil {
		return false, err
	}
	return attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}

// openLockFile creates or opens the lock file without following a reparse
// point. A symlink, junction, directory, or other non-regular leaf is
// rejected. A parent directory replaced after ensurePathContained is still
// the remaining race.
func openLockFile(path string) (*os.File, error) {
	leaf, statErr := os.Lstat(path)
	if statErr == nil {
		reparse, err := componentReparse(path)
		if err != nil {
			return nil, err
		}
		if reparse || !leaf.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is a symbolic link, junction, or non-regular file; migrate will not use it as a lock", filepath.Base(path))
		}
	} else if !isMissing(statErr) {
		return nil, statErr
	}
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pointer,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("%s is a symbolic link, junction, or non-regular file; migrate will not use it as a lock", filepath.Base(path))
	}
	return os.NewFile(uintptr(handle), path), nil
}

func lockOS(file *os.File) error {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &overlapped)
	if err != nil {
		return fmt.Errorf("could not lock the migration directory: %s", err.Error())
	}
	return nil
}

func unlockOS(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped)
}

// readFileNoFollow reads path without following a final symlink or junction.
// Ancestors are rejected by ensurePathContained before this opens the leaf.
// A parent replaced by a junction in the gap after that check is the remaining
// race; the leaf itself is opened with FILE_FLAG_OPEN_REPARSE_POINT and
// rejected when the handle is a reparse point.
func readFileNoFollow(path string) ([]byte, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pointer,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return nil, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return nil, fmt.Errorf("%s is a symbolic link or junction; migrate will not follow it", path)
	}
	var data []byte
	buffer := make([]byte, 32*1024)
	for {
		var read uint32
		err := windows.ReadFile(handle, buffer, &read, nil)
		if read > 0 {
			data = append(data, buffer[:read]...)
		}
		if errors.Is(err, windows.ERROR_HANDLE_EOF) || read == 0 {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}
