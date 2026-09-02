package analysis

import (
	"sync"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/module"
	"ahdcode/internal/source"
)

// Result is one document's analysis outcome: every diagnostic produced by
// compiling it as an entry point, grouped by the canonical path of the file
// each diagnostic actually belongs to (the entry document itself, or any
// imported sibling module the entry pulled in). A path with an empty slice
// means that file is known to be clean. Text carries the exact source each
// path was compiled from, so a caller building a diagnostic's editor range
// never needs a second, possibly-racing read of the file.
type Result struct {
	EntryPath   string
	Diagnostics map[string][]diagnostics.Diagnostic
	Text        map[string]string
}

// entry is what the Store caches per open document: its most recent
// analysis result plus the whole compiled module graph (the entry document
// and everything it transitively imports), so a query can resolve a symbol
// declared in an imported module -- not only ones declared in the entry
// document itself -- without recompiling.
type entry struct {
	result     Result
	entryModID module.ModuleID
	modules    map[module.ModuleID]*module.Module
	fileToPath map[source.FileID]string
}

// entryModule returns the compiled entry document's own module.
func (e *entry) entryModule() *module.Module {
	if e == nil {
		return nil
	}
	return e.modules[e.entryModID]
}

// moduleForFile returns the module whose compiled file has this FileID,
// within this same compile graph.
func (e *entry) moduleForFile(fileID source.FileID) *module.Module {
	if e == nil {
		return nil
	}
	for _, candidate := range e.modules {
		if candidate != nil && candidate.File.ID == fileID {
			return candidate
		}
	}
	return nil
}

// Store is the in-memory, document-aware analysis layer. It holds the
// current text of every open editor document and, for each one, the result
// of most recently analyzing it as a compilation entry point. It never
// writes a document's content to disk: unopened imports are read from the
// real filesystem, and an open document's saved-on-disk bytes are never
// consulted while it is open.
type Store struct {
	mutex          sync.Mutex
	documents      map[string]string
	entries        map[string]*entry
	workspaceRoots []string
}

// NewStore creates an empty document store.
func NewStore() *Store {
	return &Store{documents: make(map[string]string), entries: make(map[string]*entry)}
}

// text returns the open-buffer content for a canonical path, if any.
func (store *Store) text(path string) (string, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	value, ok := store.documents[path]
	return value, ok
}

// PrimaryOpenPath returns any currently open document path, used as the
// workspace-symbol search entry when the client does not tie the request to
// one document.
func (store *Store) PrimaryOpenPath() string {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	for path := range store.documents {
		return path
	}
	return ""
}

// Text returns the open-buffer content for a document path, if it is
// currently open. Callers converting an LSP Position to a byte offset (or
// back) for a request against an open document should use this -- it is
// exactly the text the client's own cursor position is relative to.
func (store *Store) Text(path string) (string, bool) {
	return store.text(canonicalPath(path))
}

// TextFor returns the exact source text targetPath was most recently
// compiled from as part of entryPath's own compile graph -- the same text a
// Location or diagnostic pointing into targetPath already describes,
// whether targetPath is an open overlay or was read straight from disk.
// Callers converting a cross-file Definition/References result's Span into
// an editor position for a file other than the one that was queried should
// use this rather than Text, which only ever knows about open overlays.
func (store *Store) TextFor(entryPath, targetPath string) (string, bool) {
	canonical := canonicalPath(entryPath)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	if cached == nil {
		return "", false
	}
	text, ok := cached.result.Text[targetPath]
	return text, ok
}

// Open registers a document's initial content and analyzes it as an entry
// point.
func (store *Store) Open(path, text string) Result {
	return store.set(path, text)
}

// Change replaces a document's content (full-document synchronization only,
// per the v0.2.0 scope) and re-analyzes it as an entry point.
func (store *Store) Change(path, text string) Result {
	return store.set(path, text)
}

// Close forgets a document's overlay content -- any other analysis that
// imports it now falls back to its real on-disk bytes -- and forgets its
// cached result. It returns the paths that previously held diagnostics
// attributed to this document's own analysis, so the caller can publish an
// empty diagnostic list for each of them.
func (store *Store) Close(path string) []string {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	defer store.mutex.Unlock()
	delete(store.documents, canonical)
	previous := store.entries[canonical]
	delete(store.entries, canonical)
	if previous == nil {
		return nil
	}
	paths := make([]string, 0, len(previous.result.Diagnostics))
	for owned := range previous.result.Diagnostics {
		paths = append(paths, owned)
	}
	return paths
}

func (store *Store) set(path, text string) Result {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	store.documents[canonical] = text
	store.mutex.Unlock()

	compiler := module.NewCompiler(overlay{store: store}, overlay{store: store})
	compiled := compiler.Compile(canonical)

	grouped := make(map[string][]diagnostics.Diagnostic)
	texts := make(map[string]string)
	fileToPath := make(map[source.FileID]string, len(compiled.Modules))
	for _, item := range compiled.Modules {
		if item == nil || item.File.ID == 0 {
			continue
		}
		fileToPath[item.File.ID] = item.File.Path
		if _, exists := grouped[item.File.Path]; !exists {
			grouped[item.File.Path] = nil
		}
		texts[item.File.Path] = item.File.Text
	}
	for _, item := range compiled.Diagnostics {
		owner, ok := fileToPath[item.Diagnostic.Span.FileID]
		if !ok {
			owner = canonical
		}
		grouped[owner] = append(grouped[owner], item.Diagnostic)
	}

	result := Result{EntryPath: canonical, Diagnostics: grouped, Text: texts}
	store.mutex.Lock()
	store.entries[canonical] = &entry{
		result:     result,
		entryModID: compiled.Entry,
		modules:    compiled.Modules,
		fileToPath: fileToPath,
	}
	store.mutex.Unlock()
	return result
}
