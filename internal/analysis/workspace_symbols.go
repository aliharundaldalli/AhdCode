package analysis

import (
	"sort"
	"strings"

	"ahdcode/internal/semantic"
)

// WorkspaceSymbol is one searchable declaration from a discoverable module.
type WorkspaceSymbol struct {
	Name        string
	Kind        semantic.SymbolKind
	Detail      string
	ModuleName  string
	Path        string
	StartOffset int
	EndOffset   int
}

// WorkspaceSymbols searches discoverable modules for declarations matching
// query. This is on-demand scanning, not a persistent index.
func (store *Store) WorkspaceSymbols(entryPath, query string) []WorkspaceSymbol {
	query = strings.TrimSpace(query)
	var results []WorkspaceSymbol
	seen := make(map[string]bool)

	addModule := func(modulePath, moduleName string) {
		key := moduleName + "\x00" + modulePath
		if seen[key] {
			return
		}
		seen[key] = true
		canonical := canonicalPath(modulePath)
		store.mutex.Lock()
		cached := store.entries[canonical]
		store.mutex.Unlock()
		if cached != nil {
			for _, symbol := range store.DocumentSymbols(modulePath) {
				if query != "" && !strings.Contains(strings.ToLower(symbol.Name), strings.ToLower(query)) {
					continue
				}
				results = append(results, WorkspaceSymbol{
					Name:       symbol.Name,
					Kind:       symbol.Kind,
					Detail:     symbol.Detail,
					ModuleName: moduleName,
					Path:       modulePath,
					StartOffset: symbol.Span.Start.Offset,
					EndOffset:   symbol.Span.End.Offset,
				})
			}
			return
		}
		iface := store.moduleInterfaceAt(modulePath)
		if iface == nil {
			return
		}
		for _, name := range iface.ExportNames {
			if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
				continue
			}
			exported := iface.Exports[name]
			if exported == nil || exported.Confidential {
				continue
			}
			results = append(results, WorkspaceSymbol{
				Name:       name,
				Kind:       exported.Kind,
				Detail:     renderHover(exported),
				ModuleName: moduleName,
				Path:       modulePath,
			})
		}
	}

	canonical := canonicalPath(entryPath)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	if cached != nil {
		entryModule := cached.entryModule()
		if entryModule != nil && entryModule.Interface != nil {
			addModule(entryModule.File.Path, entryModule.Source.Name)
		}
	}
	for _, discovered := range store.discoverUserModules(entryPath) {
		addModule(discovered.Path, discovered.Name)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		return results[i].ModuleName < results[j].ModuleName
	})
	return results
}
