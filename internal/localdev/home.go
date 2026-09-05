// Package localdev holds the per-user, local-development-only registries
// AhdCode v0.19 introduces: the route registry behind automatic `.test`
// hostnames, and the database registry behind AhdDataStudio discovery.
//
// This is deliberately not a project package manager and not a service
// manager. Nothing here is fetched, nothing is executed, nothing is shared
// between machines, and nothing recorded here is a secret: a registry entry
// is local development metadata (a hostname, a loopback port, a file path)
// that exists so the toolchain can present a clean URL and a visible
// database without the user hand-editing configuration.
//
// Every file this package owns lives under one per-user directory, is
// written atomically, and is readable only by its owner.
package localdev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HomeEnvKey overrides the per-user local-development directory. It exists
// for tests and for isolated environments; ordinary use never sets it.
const HomeEnvKey = "AHDCODE_LOCAL_HOME"

const (
	routesFileName    = "routes.json"
	databasesFileName = "databases.json"
)

// Home resolves the directory holding this user's local-development state.
//
// It follows the same convention internal/build/runcache.go already uses for
// per-user toolchain state -- an OS-provided per-user location plus an
// "ahdcode" component -- except that a registry is durable configuration
// rather than a rebuildable cache, so it uses the config location: a purged
// cache directory must never silently drop a user's registered databases.
//
// The directory is created 0700 on first use: it names the user's projects
// and their local ports, which is nobody else's business on a shared machine.
func Home() (string, error) {
	if override := strings.TrimSpace(os.Getenv(HomeEnvKey)); override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("%s must be an absolute path", HomeEnvKey)
		}
		return ensureDir(filepath.Clean(override))
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate a per-user configuration directory: %v", err)
	}
	return ensureDir(filepath.Join(base, "ahdcode"))
}

func ensureDir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create %s: %v", dir, err)
	}
	return dir, nil
}

// RoutesPath is the absolute path of the local route registry.
func RoutesPath() (string, error) { return childOfHome(routesFileName) }

// DatabasesPath is the absolute path of the local database registry.
func DatabasesPath() (string, error) { return childOfHome(databasesFileName) }

func childOfHome(name string) (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, name), nil
}
