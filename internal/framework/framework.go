// Package framework holds AhdCode's bundled first-party source modules: the
// Web framework is written in AhdCode itself and embedded in the compiler,
// not shipped as .ahd files a built application has to find at runtime.
//
// Resolution is deterministic and entirely offline. There is no registry, no
// manifest, no lockfile, and no remote fetch: every module below is compiled
// from the bytes embedded in this package, in the same frontend pass as the
// application's own files, and is then lowered and generated into the single
// self-contained executable like any other module. Removing the compiler
// repository, or the sources/ directory, from a machine that already has a
// built binary changes nothing about that binary.
package framework

import (
	"embed"
	"sort"
	"strings"
)

//go:embed sources/*.ahd
var sources embed.FS

// public names the modules an application may name in `bring`. Everything
// else under sources/ is framework-internal: only another framework module
// can reach it, so the framework's own decomposition never becomes part of
// the language's public surface.
var public = map[string]bool{"Web": true}

// facades names the framework modules whose named imports are re-exported
// through their own module interface. This is what lets `bring Web` hand an
// application the existing HTTP.Request / HTML.HTMLNode identities under the
// Web namespace instead of forcing a second, incompatible set of types: a
// facade re-exports the very symbols it imported, never a copy of them.
var facades = map[string]bool{"Web": true}

// Has reports whether name is a bundled first-party module at all.
func Has(name string) bool {
	_, err := sources.ReadFile(sourcePath(name))
	return err == nil
}

// IsPublic reports whether an ordinary application file may `bring name`.
func IsPublic(name string) bool { return public[name] && Has(name) }

// IsFacade reports whether name re-exports the symbols it imports.
func IsFacade(name string) bool { return facades[name] && Has(name) }

// Source returns the embedded AhdCode text for one bundled module.
func Source(name string) (string, bool) {
	data, err := sources.ReadFile(sourcePath(name))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// PublicModuleNames lists, sorted, the bundled modules an application may
// bring. Tooling derives its module list from this rather than hardcoding a
// catalog of the framework's API.
func PublicModuleNames() []string {
	names := make([]string, 0, len(public))
	for name := range public {
		if Has(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// ModuleNames lists every bundled module, public or internal, sorted.
func ModuleNames() []string {
	entries, err := sources.ReadDir("sources")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ahd") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".ahd"))
	}
	sort.Strings(names)
	return names
}

// ModuleID is the canonical compile-time identity of a bundled module. The
// framework: prefix keeps it distinct from both builtin: and file: identities
// for every consumer that keys on module identity.
func ModuleID(name string) string { return "framework:" + name }

// VirtualPath is the path diagnostics show for a bundled module. It is
// deliberately not a real filesystem path: nothing on disk needs to exist for
// the module to compile, and `ahdcode dev` must never try to watch it.
func VirtualPath(name string) string { return "<ahdcode>/" + name + ".ahd" }

// IsVirtualPath reports whether a path came from a bundled module rather than
// from the user's source tree.
func IsVirtualPath(path string) bool { return strings.HasPrefix(path, "<ahdcode>/") }

func sourcePath(name string) string {
	if name == "" || strings.ContainsAny(name, "/\\.") {
		return "sources/\x00"
	}
	return "sources/" + name + ".ahd"
}
