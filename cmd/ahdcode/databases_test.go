package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStudioHostsMapsLoopback(t *testing.T) {
	for _, tc := range []struct {
		name, hosts string
		want        bool
	}{
		{"missing", "127.0.0.1 localhost\n", false},
		{"comment", "# 127.0.0.1 ahddatabasestudio.test\n", false},
		{"aliases", "127.0.0.1\tlocalhost AHDDATABASESTUDIO.TEST # local\n", true},
		{"ipv6 only", "::1 ahddatabasestudio.test\n", false},
		{"remote", "192.0.2.1 ahddatabasestudio.test\n", false},
		{"conflict", "127.0.0.1 ahddatabasestudio.test\n::1 ahddatabasestudio.test\n", false},
		{"suffix", "127.0.0.1 ahddatabasestudio.test.invalid\n", false},
		{"absolute name", "127.0.0.1 ahddatabasestudio.test.\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := studioHostsMapsLoopback(strings.NewReader(tc.hosts)); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestStudioEnvBootstrapPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("AHD_DATA_LANG=en\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureStudioEnv(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %v", info.Mode())
	}
	existing := []byte("AHD_DATA_LANG=tr\n")
	if err := os.WriteFile(path, existing, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureStudioEnv(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(existing) {
		t.Fatalf("existing env changed: %v", err)
	}
}
