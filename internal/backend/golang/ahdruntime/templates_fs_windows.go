//go:build windows

package ahdruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	ahdTemplateKernel32          = syscall.NewLazyDLL("kernel32.dll")
	ahdTemplateGetFileAttributes = ahdTemplateKernel32.NewProc("GetFileAttributesW")
)

func ahdTemplateMissing(err error) bool {
	return os.IsNotExist(err) || errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(err, syscall.ERROR_PATH_NOT_FOUND)
}

func ahdTemplateComponentReparse(path string) (bool, error) {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attributes, _, callErr := ahdTemplateGetFileAttributes.Call(uintptr(unsafe.Pointer(pointer)))
	if attributes == uintptr(syscall.INVALID_FILE_ATTRIBUTES) {
		if callErr != nil {
			return false, callErr
		}
		return false, syscall.ERROR_FILE_NOT_FOUND
	}
	return attributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}

// ahdTemplateReadFile opens the leaf without following reparse points and
// checks the opened handle before reading. Ancestor checks are path-based.
func ahdTemplateReadFile(path string) ([]byte, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	handle := syscall.Handle(file.Fd())
	kind, err := syscall.GetFileType(handle)
	if err != nil {
		return nil, err
	}
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &info); err != nil {
		return nil, err
	}
	if kind != syscall.FILE_TYPE_DISK ||
		info.FileAttributes&(syscall.FILE_ATTRIBUTE_REPARSE_POINT|syscall.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		return nil, fmt.Errorf("%s is a reparse point or non-regular file", filepath.Base(path))
	}
	return ahdTemplateReadLimited(file, ahdTemplateMaxBytes)
}
