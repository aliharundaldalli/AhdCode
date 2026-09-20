package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"ahdcode/internal/semantic"
)

const maxWorkspaceSourceFiles = 1024

// workspaceEntries returns the cached open-document snapshots plus bounded,
// on-demand compiler snapshots for other .ahd files below the initialized
// workspace roots. It is deliberately query-time and in-memory: there is no
// persistent database and no text-based symbol matching.
func (store *Store) workspaceEntries(entryPath string) []*entry {
	canonicalEntry := canonicalPath(entryPath)
	store.mutex.Lock()
	cached := make(map[string]*entry, len(store.entries))
	for path, item := range store.entries {
		cached[path] = item
	}
	store.mutex.Unlock()

	paths := make(map[string]bool, len(cached))
	for path := range cached {
		paths[path] = true
	}
	for _, path := range store.workspaceSourceFiles(canonicalEntry) {
		paths[path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)

	result := make([]*entry, 0, len(ordered))
	for _, path := range ordered {
		item := cached[path]
		if item == nil {
			item = store.compileEntry(path)
		}
		if item != nil && item.entryModule() != nil {
			result = append(result, item)
		}
	}
	return result
}

func (store *Store) workspaceSourceFiles(entryPath string) []string {
	roots := store.workspaceRootsFor(entryPath)
	seen := make(map[string]bool)
	var paths []string
	for _, root := range roots {
		// A URI such as file:///main.ahd has "/" as its entry directory in
		// protocol/unit tests. Scanning the filesystem root would be both
		// surprising and unbounded in practice; an explicit project root must
		// be narrower than the volume root to participate in workspace queries.
		if filepath.Clean(root) == string(filepath.Separator) {
			continue
		}
		if len(paths) >= maxWorkspaceSourceFiles {
			break
		}
		_ = filepath.WalkDir(root, func(path string, item os.DirEntry, err error) error {
			if err != nil || item == nil {
				return nil
			}
			if item.IsDir() {
				name := item.Name()
				if name == ".git" || name == ".codex" || name == "node_modules" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if len(paths) >= maxWorkspaceSourceFiles || filepath.Ext(path) != ".ahd" {
				return nil
			}
			canonical := canonicalPath(path)
			if !seen[canonical] {
				seen[canonical] = true
				paths = append(paths, canonical)
			}
			return nil
		})
	}
	return paths
}

// declarationIdentity returns the canonical source identity for a resolved
// symbol. It lives here so references and rename use exactly the same rule.
func declarationIdentity(e *entry, symbol *semantic.Symbol) string {
	if e == nil || symbol == nil {
		return ""
	}
	declaration := e.declarationSymbol(symbol)
	if declaration == nil {
		return ""
	}
	path := e.fileToPath[declaration.Span.FileID]
	if path == "" {
		// Built-in and interface-only symbols have no source identity.
		return ""
	}
	owner := ""
	if declaration.OwnerClass != nil {
		owner = declaration.OwnerClass.Name
	}
	return canonicalPath(path) + "\x00" + owner + "\x00" + declaration.Name + "\x00" +
		itoa(declaration.Span.Start.Offset) + "\x00" + itoa(declaration.Span.End.Offset)
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
