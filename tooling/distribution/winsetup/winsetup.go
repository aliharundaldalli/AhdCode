// Package winsetup holds the Windows setup program's platform-independent
// decisions: which payload entries are safe to extract, how the stable
// launcher finds the active version, and how the user's PATH is edited.
//
// The rules live here, away from the Win32 calls, so they are exercised by
// ordinary `go test` on any development machine rather than only on Windows.
package winsetup

import (
	"errors"
	"path/filepath"
	"strings"
)

// LauncherPointerName is the file under the installation root that records the
// active version. The stable launcher reads it on every invocation, so an
// upgrade switches versions without touching PATH.
const LauncherPointerName = "current.txt"

// ErrUnsafeEntry reports a payload entry that must not be extracted.
var ErrUnsafeEntry = errors.New("unsafe payload entry")

// CheckEntryName rejects payload names that would escape the staging
// directory. Windows accepts both separators, so both are inspected, and a
// drive-relative or rooted name is refused outright.
func CheckEntryName(name string) error {
	if name == "" {
		return ErrUnsafeEntry
	}
	if strings.ContainsAny(name, "\x00") {
		return ErrUnsafeEntry
	}
	normalized := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(normalized, "/") {
		return ErrUnsafeEntry
	}
	if len(name) >= 2 && name[1] == ':' {
		return ErrUnsafeEntry
	}
	for _, element := range strings.Split(normalized, "/") {
		if element == "" || element == "." || element == ".." {
			return ErrUnsafeEntry
		}
	}
	return nil
}

// CheckVersion rejects a release identity that could not name a directory.
func CheckVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" || version == "." || version == ".." {
		return errors.New("invalid release identity")
	}
	if strings.ContainsAny(version, `/\:*?"<>|`) {
		return errors.New("invalid release identity")
	}
	return nil
}

// ReadPointer interprets the contents of the active-version pointer file.
func ReadPointer(content []byte) (string, error) {
	version := strings.TrimSpace(string(content))
	if err := CheckVersion(version); err != nil {
		return "", err
	}
	return version, nil
}

// ActiveExecutable names the ahdcode.exe that the stable launcher must run for
// a given installation root and recorded version.
func ActiveExecutable(root, version string) string {
	return filepath.Join(root, "versions", version, "bin", "ahdcode.exe")
}

// LauncherDirectory names the fixed folder that goes on PATH. It never changes
// between versions, so PATH is written once for the life of the installation.
func LauncherDirectory(root string) string {
	return filepath.Join(root, "bin")
}

// SplitPath separates a raw PATH value into entries, preserving them exactly.
// Windows tolerates empty segments, and dropping them would silently rewrite
// an unrelated part of the user's PATH, so they are kept.
func SplitPath(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ";")
}

// samePathEntry compares two PATH entries the way Windows resolves them:
// case-insensitively and ignoring surrounding quotes and trailing separators.
func samePathEntry(a, b string) bool {
	normalize := func(value string) string {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"`)
		value = strings.ReplaceAll(value, "/", `\`)
		value = strings.TrimRight(value, `\`)
		return strings.ToLower(value)
	}
	return normalize(a) != "" && normalize(a) == normalize(b)
}

// HasPathEntry reports whether directory is already on the raw PATH value.
// expand resolves %NAME% references so an entry written as
// %LOCALAPPDATA%\AhdCode\bin is recognized; it may be nil.
func HasPathEntry(raw, directory string, expand func(string) string) bool {
	for _, entry := range SplitPath(raw) {
		if samePathEntry(entry, directory) {
			return true
		}
		if expand != nil && entry != "" {
			if samePathEntry(expand(entry), directory) {
				return true
			}
		}
	}
	return false
}

// AddPathEntry returns the raw PATH value with directory present exactly once.
// Unrelated entries are preserved byte for byte and order is kept; the new
// entry is appended so it can never shadow a program the user already has.
// The boolean reports whether the value changed.
func AddPathEntry(raw, directory string, expand func(string) string) (string, bool) {
	if directory == "" {
		return raw, false
	}
	if HasPathEntry(raw, directory, expand) {
		return raw, false
	}
	if raw == "" {
		return directory, true
	}
	if strings.HasSuffix(raw, ";") {
		return raw + directory, true
	}
	return raw + ";" + directory, true
}

// RemovePathEntry returns the raw PATH value without directory, preserving
// every unrelated entry. The boolean reports whether the value changed.
func RemovePathEntry(raw, directory string, expand func(string) string) (string, bool) {
	entries := SplitPath(raw)
	kept := make([]string, 0, len(entries))
	removed := false
	for _, entry := range entries {
		match := samePathEntry(entry, directory)
		if !match && expand != nil && entry != "" {
			match = samePathEntry(expand(entry), directory)
		}
		if match {
			removed = true
			continue
		}
		kept = append(kept, entry)
	}
	if !removed {
		return raw, false
	}
	return strings.Join(kept, ";"), true
}
