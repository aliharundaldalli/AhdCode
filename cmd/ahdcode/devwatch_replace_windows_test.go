//go:build windows

package main

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// replaceFile is the save editors use on Windows: write a temporary file in
// the same directory, then replace the destination. MoveFileEx can replace
// an existing file; os.Rename on Windows cannot.
func replaceFile(t *testing.T, path, content string) {
	t.Helper()
	temporary := path + ".saving"
	if err := os.WriteFile(temporary, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	from, err := windows.UTF16PtrFromString(temporary)
	if err != nil {
		t.Fatal(err)
	}
	to, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil {
		t.Fatal(err)
	}
}
