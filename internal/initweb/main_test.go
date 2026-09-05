package initweb

import (
	"os"
	"testing"

	"ahdcode/internal/localdev"
)

// Every test in this package runs against a throwaway local-development home.
// initweb now registers a created SQLite database in the per-user registry,
// and a test suite must never write into the registry of whoever is running
// it -- a passing test run should leave a developer's own registered
// databases exactly as it found them.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "ahdcode-initweb-home-*")
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
