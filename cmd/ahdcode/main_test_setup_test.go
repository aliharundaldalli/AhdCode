package main

import (
	"os"
	"testing"

	"ahdcode/internal/localdev"
)

// The whole CLI suite runs against a throwaway local-development home.
// `ahdcode dev`, `ahdcode databases`, and `ahdcode init web admin` all write
// to the per-user registries now, and a test run must leave the registries of
// whoever is running it exactly as it found them.
func TestMain(m *testing.M) {
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
