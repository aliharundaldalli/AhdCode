//go:build !windows

package main

import (
	"os"
	"testing"
)

// replaceFile writes a temporary sibling and renames it onto path. On Unix
// that rename replaces the destination.
func replaceFile(t *testing.T, path, content string) {
	t.Helper()
	temporary := path + ".saving"
	if err := os.WriteFile(temporary, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporary, path); err != nil {
		t.Fatal(err)
	}
}
