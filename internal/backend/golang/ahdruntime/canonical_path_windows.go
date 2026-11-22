package ahdruntime

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var ahdFinalPathName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFinalPathNameByHandleW")

func ahdCanonicalExistingPath(path string) (string, error) {
	// The standard path walker may leave a junction in an otherwise
	// resolved directory name. Read the physical path of the opened handle
	// for both the root and the candidate. Callers enforce containment.
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	buffer := make([]uint16, 512)
	for {
		length, _, callErr := ahdFinalPathName.Call(file.Fd(), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), 0)
		if length == 0 {
			return "", callErr
		}
		if length >= uintptr(len(buffer)) {
			buffer = make([]uint16, length+1)
			continue
		}
		final := syscall.UTF16ToString(buffer[:length])
		if strings.HasPrefix(final, `\\?\UNC\`) {
			return `\\` + final[len(`\\?\UNC\`):], nil
		}
		final = strings.TrimPrefix(final, `\\?\`)
		if filepath.IsAbs(final) && len(filepath.VolumeName(final)) == 2 {
			return final, nil
		}
		return "", fs.ErrInvalid
	}
}
