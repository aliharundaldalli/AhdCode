// Package analysis is the document-aware compiler analysis layer for
// AhdCode tooling (the v0.2.0 language server foundation). It knows how to
// compile an in-memory, possibly-unsaved document -- resolving any sibling
// modules it imports against other open documents first, then the real
// filesystem -- and to expose the resulting diagnostics and hover facts.
//
// This package has no knowledge of JSON-RPC, LSP wire types, or editor
// concepts. It consumes the existing AhdCode compiler frontend
// (internal/lexer, internal/parser, internal/semantic, via
// internal/module.Compiler) exactly as internal/repl already does for its
// own in-memory entry source; it never reimplements lexing, parsing, or
// type checking, and it never writes an open document's content to disk.
package analysis

import (
	"path/filepath"

	"ahdcode/internal/module"
)

// CanonicalPath normalizes a filesystem path the same way internal/module's
// own (unexported) canonicalFilePath does: absolute, symlink-resolved where
// possible, cleaned. Overlay lookups use this exact normalization so a
// document opened under one spelling of its path still matches the
// module.SourceIdentity.Path the compiler resolves for it. It is exported
// so a caller (the LSP layer) can compute the same key a Result's EntryPath
// already uses, e.g. to track per-entry state across Store calls, without
// duplicating this normalization.
func CanonicalPath(value string) string {
	return canonicalPath(value)
}

func canonicalPath(value string) string {
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return filepath.Clean(value)
	}
	if evaluated, evaluateError := filepath.EvalSymlinks(absolute); evaluateError == nil {
		absolute = evaluated
	}
	return filepath.Clean(absolute)
}

// overlay implements module.ModuleResolver and module.SourceLoader by
// delegating entirely to the real filesystem resolver/loader, except that
// Load first checks the document store for an open buffer's current text.
// This generalizes internal/repl's single-entry overlayWorkspace to an
// arbitrary set of open documents: any module whose canonical path matches
// an open document uses that document's in-memory text -- whether it is the
// analysis entry point or an imported sibling that also happens to be open
// -- and every other module still reads real file bytes from disk. No
// document is ever written back to disk by this type.
type overlay struct {
	store    *Store
	resolver module.FileResolver
	loader   module.FileLoader
}

func (o overlay) CanonicalEntry(entryPath string) (module.SourceIdentity, error) {
	return o.resolver.CanonicalEntry(entryPath)
}

func (o overlay) Resolve(importer module.SourceIdentity, moduleName string) (module.SourceIdentity, error) {
	return o.resolver.Resolve(importer, moduleName)
}

func (o overlay) Load(identity module.SourceIdentity) (string, error) {
	if text, ok := o.store.text(canonicalPath(identity.Path)); ok {
		return text, nil
	}
	return o.loader.Load(identity)
}
