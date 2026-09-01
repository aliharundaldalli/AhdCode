package lsp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// URIToPath converts a textDocument URI into a real filesystem path. Only
// the "file" scheme is supported in v0.2.0; any other scheme (untitled:,
// vscode-notebook-cell:, ...) is reported as an error so callers can ignore
// that document cleanly rather than treating URI text as a filesystem path.
//
// This always goes through net/url.Parse -- never manual "file://" string
// trimming -- so percent-encoding (spaces, Unicode path segments) is
// decoded correctly, and a Windows drive-letter path (file:///C:/Users/...)
// has its extra leading slash stripped.
func URIToPath(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid document URI %q: %w", uri, err)
	}
	if parsed.Scheme != "file" {
		return "", fmt.Errorf("unsupported document URI scheme %q", parsed.Scheme)
	}
	path := parsed.Path
	if isWindowsDriveURIPath(path) {
		path = path[1:]
	}
	return filepath.FromSlash(path), nil
}

// PathToURI converts a real filesystem path into a file:// URI, letting
// net/url handle percent-encoding for spaces and Unicode path segments.
//
// A Windows drive-letter path (e.g. "C:\Users\..." or "C:/Users/...") is
// recognized and slash-normalized explicitly here, rather than through
// filepath.ToSlash: that function only rewrites the host OS's own separator,
// so it silently leaves backslashes untouched when this process itself is
// not running on Windows -- exactly the case a cross-platform test for
// Windows paths needs to cover.
func PathToURI(path string) string {
	slashed := path
	if isASCIILetter(safeByteAt(path, 0)) && safeByteAt(path, 1) == ':' {
		slashed = strings.ReplaceAll(path, "\\", "/")
	} else {
		slashed = filepath.ToSlash(path)
	}
	if isWindowsDrivePath(slashed) {
		slashed = "/" + slashed
	}
	target := url.URL{Scheme: "file", Path: slashed}
	return target.String()
}

func safeByteAt(value string, index int) byte {
	if index < 0 || index >= len(value) {
		return 0
	}
	return value[index]
}

// isWindowsDriveURIPath reports whether a URI's decoded path component
// looks like "/C:/..." -- the form file:// URIs use for a Windows drive
// letter, which needs its leading slash stripped to become "C:/...".
func isWindowsDriveURIPath(path string) bool {
	return len(path) >= 3 && path[0] == '/' && isASCIILetter(path[1]) && path[2] == ':'
}

// isWindowsDrivePath reports whether a slash-normalized filesystem path
// starts with a drive letter, e.g. "C:/Users/...".
func isWindowsDrivePath(path string) bool {
	return len(path) >= 2 && isASCIILetter(path[0]) && path[1] == ':' && (len(path) == 2 || path[2] == '/')
}

func isASCIILetter(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z')
}
