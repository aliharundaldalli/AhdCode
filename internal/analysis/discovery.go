package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ahdcode/internal/framework"
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
)

// discoveredModule is one user .ahd module found by scanning workspace roots.
type discoveredModule struct {
	Name string
	Path string
}

// SetWorkspaceRoots records workspace folder paths from the LSP initialize
// request. Discovery scans these directories plus each entry document's own
// directory for sibling .ahd modules.
func (store *Store) SetWorkspaceRoots(roots []string) {
	canonical := make([]string, 0, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		canonical = append(canonical, canonicalPath(root))
	}
	store.mutex.Lock()
	store.workspaceRoots = canonical
	store.mutex.Unlock()
}

func (store *Store) workspaceRootsFor(entryPath string) []string {
	store.mutex.Lock()
	roots := append([]string(nil), store.workspaceRoots...)
	store.mutex.Unlock()
	entryDir := filepath.Dir(canonicalPath(entryPath))
	seen := make(map[string]bool)
	var out []string
	add := func(path string) {
		path = canonicalPath(path)
		if path != "" && !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	add(entryDir)
	for _, root := range roots {
		add(root)
	}
	return out
}

// discoverUserModules lists every user .ahd module under the bounded scan
// roots relevant to entryPath. Standard modules are excluded; results are
// sorted by module name.
func (store *Store) discoverUserModules(entryPath string) []discoveredModule {
	standards := semantic.StandardModuleInterfaces()
	seen := make(map[string]discoveredModule)
	for _, root := range store.workspaceRootsFor(entryPath) {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ahd") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".ahd")
			// A file whose name matches a standard or bundled first-party
			// module can never be the module a bring resolves to -- those
			// names are reserved and win ahead of the filesystem -- so
			// offering it here would only mislead.
			if name == "" || standards[name] != nil || framework.IsPublic(name) {
				continue
			}
			path := canonicalPath(filepath.Join(root, entry.Name()))
			if _, exists := seen[name]; !exists {
				seen[name] = discoveredModule{Name: name, Path: path}
			}
		}
	}
	out := make([]discoveredModule, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// moduleInterfaceAt compiles one module path on demand (respecting open
// document overlays) and returns its public interface, or nil on failure.
func (store *Store) moduleInterfaceAt(modulePath string) *semantic.ModuleInterface {
	canonical := canonicalPath(modulePath)
	store.mutex.Lock()
	for entryPath, cached := range store.entries {
		if cached == nil {
			continue
		}
		for _, candidate := range cached.modules {
			if candidate != nil && candidate.File.Path == canonical && candidate.Interface != nil {
				store.mutex.Unlock()
				return candidate.Interface
			}
		}
		_ = entryPath
	}
	store.mutex.Unlock()

	compiler := module.NewCompiler(overlay{store: store}, overlay{store: store})
	compiled := compiler.Compile(canonical)
	if moduleItem := compiled.Modules[compiled.Entry]; moduleItem != nil {
		return moduleItem.Interface
	}
	return nil
}
