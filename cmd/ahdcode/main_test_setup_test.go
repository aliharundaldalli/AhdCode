package main

import (
	"os"
	"os/exec"
	"testing"

	"ahdcode/internal/localdev"
)

// The whole CLI suite runs against a throwaway local-development home.
// `ahdcode dev`, `ahdcode databases`, and `ahdcode init web admin` all write
// to the per-user registries now, and a test run must leave the registries of
// whoever is running it exactly as it found them.
func TestMain(m *testing.M) {
	if entry := os.Getenv("AHDCODE_TEST_CHECK_ENTRY"); entry != "" {
		if _, err := exec.LookPath("go"); err == nil {
			panic("Go is visible on PATH")
		}
		os.Exit(run([]string{"check", entry, "--format", "json"}))
	}
	home, err := os.MkdirTemp("", "ahdcode-cli-home-*")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv(localdev.HomeEnvKey, home); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}
