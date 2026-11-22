//go:build !windows

package main

import (
	"os"
	"testing"
)

func prepareStudioEnvPermissionFixture(t *testing.T, dir string) {}

func studioEnvPermissionFingerprint(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("env mode %v, want 0600", info.Mode())
	}
	return info.Mode().Perm().String()
}
