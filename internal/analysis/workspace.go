package analysis

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"ahdcode/internal/semantic"
)

type workspaceBudget struct {
	directories, entries, files int
}

var workspaceScanBudget = workspaceBudget{directories: 512, entries: 8192, files: 1024}

// workspaceEntries returns the cached open-document snapshots plus bounded,
// on-demand compiler snapshots for other .ahd files below the initialized
// workspace roots. It is deliberately query-time and in-memory: there is no
// persistent database and no text-based symbol matching.
func (store *Store) workspaceEntries(entryPath string, entrySnapshot *entry, requireComplete bool) ([]*entry, bool) {
	canonicalEntry := canonicalPath(entryPath)
	store.mutex.Lock()
	revision := store.documentRevision
	cached := make(map[string]*entry, len(store.entries))
	for path, item := range store.entries {
		cached[path] = item
	}
	store.mutex.Unlock()
	cached[canonicalEntry] = entrySnapshot

	paths := make(map[string]bool, len(cached))
	for path := range cached {
		paths[path] = true
	}
	discovered, complete := workspaceSourceFiles(store.workspaceRootsFor(canonicalEntry), workspaceScanBudget)
	if requireComplete && !complete {
		return nil, false
	}
	for _, path := range discovered {
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
		if item == nil || item.documentRevision != revision {
			item = store.compileEntry(path)
		}
		if item != nil && item.entryModule() != nil {
			result = append(result, item)
		} else {
			complete = false
		}
	}
	return result, complete
}

// Workspace requests can resolve the same declaration across several open
// documents. Refresh a snapshot after another document opens, changes or
// closes so its spans and identities do not refer to superseded overlay text.
func (store *Store) workspaceEntry(path string) *entry {
	store.mutex.Lock()
	item := store.entries[path]
	revision := store.documentRevision
	store.mutex.Unlock()
	if item != nil && item.documentRevision != revision {
		return store.compileEntry(path)
	}
	return item
}

type workspaceScan struct {
	budget      workspaceBudget
	directories map[string]bool
	files       map[string]bool
	paths       []string
	incomplete  bool
}

func workspaceSourceFiles(roots []string, budget workspaceBudget) ([]string, bool) {
	scan := workspaceScan{budget: budget, directories: make(map[string]bool), files: make(map[string]bool)}
	for _, root := range roots {
		if isWorkspaceVolumeRoot(root) || scan.directories[root] {
			continue
		}
		info, err := os.Lstat(root)
		if errors.Is(err, fs.ErrNotExist) {
			// An unsaved document can live in a directory that does not exist.
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			scan.incomplete = true
			continue
		}
		tree, err := os.OpenRoot(root)
		if err != nil {
			scan.incomplete = true
			continue
		}
		opened, err := tree.Stat(".")
		canContinue := true
		if err != nil || !os.SameFile(info, opened) {
			scan.incomplete = true
		} else {
			canContinue = scan.walk(tree, ".")
		}
		_ = tree.Close()
		if !canContinue {
			scan.incomplete = true
			break
		}
	}
	return scan.paths, !scan.incomplete
}

// WalkDir sorts by reading the whole directory first. Read at most one entry
// beyond the remaining budget and omit an oversized directory rather than
// picking an arbitrary subset from the filesystem's enumeration order.
func (scan *workspaceScan) walk(tree *os.Root, directory string) bool {
	path := filepath.Join(tree.Name(), directory)
	if scan.directories[path] {
		return true
	}
	if scan.budget.directories == 0 {
		return false
	}
	scan.budget.directories--
	scan.directories[path] = true

	info, err := tree.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		scan.incomplete = true
		return true
	}
	handle, err := tree.Open(directory)
	if err != nil {
		scan.incomplete = true
		return true
	}
	opened, err := handle.Stat()
	if err != nil || !os.SameFile(info, opened) {
		_ = handle.Close()
		scan.incomplete = true
		return true
	}
	items, err := handle.ReadDir(scan.budget.entries + 1)
	_ = handle.Close()
	if len(items) > scan.budget.entries {
		return false
	}
	scan.budget.entries -= len(items)
	if err != nil && !errors.Is(err, io.EOF) {
		scan.incomplete = true
		return true
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name() < items[j].Name() })
	for _, item := range items {
		child := filepath.Join(directory, item.Name())
		if item.Type()&os.ModeSymlink != 0 {
			continue
		}
		if item.IsDir() {
			switch item.Name() {
			case ".git", ".codex", "node_modules", "vendor":
				continue
			}
			if !scan.walk(tree, child) {
				return false
			}
			continue
		}
		if !item.Type().IsRegular() || filepath.Ext(child) != ".ahd" {
			continue
		}
		candidate := filepath.Join(tree.Name(), child)
		if scan.files[candidate] {
			continue
		}
		if scan.budget.files == 0 {
			return false
		}
		scan.budget.files--
		scan.files[candidate] = true
		scan.paths = append(scan.paths, candidate)
	}
	return true
}

func isWorkspaceVolumeRoot(path string) bool {
	path = filepath.Clean(path)
	return filepath.IsAbs(path) && filepath.Dir(path) == path
}

type symbolDeclarationKey struct {
	path, owner, name string
	start, end        int
}

// declarationIdentity joins snapshots by the resolved declaration's source
// path and offsets. Paths in fileToPath already come from module resolution;
// there is no reason to resolve each AST occurrence against the disk again.
func declarationIdentity(e *entry, declaration *semantic.Symbol) symbolDeclarationKey {
	if declaration == nil {
		return symbolDeclarationKey{}
	}
	path := e.fileToPath[declaration.Span.FileID]
	if path == "" {
		// Built-in and interface-only symbols have no source identity.
		return symbolDeclarationKey{}
	}
	owner := ""
	if declaration.OwnerClass != nil {
		owner = declaration.OwnerClass.Name
	}
	return symbolDeclarationKey{path, owner, declaration.Name, declaration.Span.Start.Offset, declaration.Span.End.Offset}
}

type sourceOccurrence struct {
	path       string
	start, end int
}
