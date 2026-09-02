package analysis

import (
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
)

// Location identifies a span of source in a specific file -- the file is
// named by its canonical path rather than a source.FileID, since a FileID is
// only meaningful within the single compile that produced it.
type Location struct {
	Path string
	Span source.Span
}

// Definition answers a go-to-definition request at the given byte offset
// within the most recently analyzed snapshot of path. It resolves across the
// whole compile graph (the document plus everything it transitively
// imports), so a use of an imported symbol correctly jumps into the module
// that actually declares it. It reports ok=false whenever there is no
// confidently resolved symbol at that position, or the symbol has no
// AhdCode-source declaration to jump to (a builtin, for instance).
func (store *Store) Definition(path string, offset int) (Location, bool) {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()

	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return Location{}, false
	}
	node := findNodeAtOffset(entryModule.Parsed.Program, offset)
	if node == nil {
		return Location{}, false
	}
	symbol, ok := entryModule.Semantic.ResolvedSymbols[node]
	if !ok || symbol == nil || symbol.Builtin {
		return Location{}, false
	}
	symbol = cached.declarationSymbol(symbol)
	if symbol == nil {
		return Location{}, false
	}
	declarationPath, ok := cached.fileToPath[symbol.Span.FileID]
	if !ok {
		return Location{}, false
	}
	return Location{Path: declarationPath, Span: symbol.Span}, true
}

// declarationSymbol resolves a symbol reached through ResolvedSymbols to the
// one carrying its real declaration-site Span. A symbol reached across a
// module boundary (a namespace member, or a member of a Class declared in
// another module) is a clone built by BuildModuleInterface, which -- by
// design, so that a module's public interface stays independent of any one
// compile's AST/positions -- never copies Span or Declaration. Since this
// query answers a single request against one whole-graph compile, the real
// declaration-site symbol equivalent is still reachable: it lives in the
// owning module's own Semantic.Symbols, keyed by the same OriginModuleID,
// Name, and (for a Class member) the same OwnerClass identity.
//
// A Class attribute has one further wrinkle even within its own module: the
// analyzer records two distinct Symbol objects at the same syntactic
// position -- a ParameterSymbol (for analyzing the structure's own body) and
// the MemberSymbol actually stored in the Class's Members map (what every
// `receiver.attribute` access and cross-module clone are built from).
// Clicking directly on the attribute's declaration resolves the former, so
// it is upgraded here to the latter by matching the identical Span both
// were built from, keeping every caller's view of "this attribute's
// declaration symbol" consistent regardless of which one ResolvedSymbols
// happened to hand back.
func (e *entry) declarationSymbol(symbol *semantic.Symbol) *semantic.Symbol {
	if symbol.Span.FileID == 0 {
		owner := e.modules[module.ModuleID(symbol.OriginModuleID)]
		if owner == nil {
			return nil
		}
		for _, candidate := range owner.Semantic.Symbols {
			if candidate == nil || candidate.Span.FileID == 0 || candidate.Name != symbol.Name {
				continue
			}
			if candidate.OwnerClass == symbol.OwnerClass {
				return candidate
			}
		}
		return nil
	}
	if symbol.Kind == semantic.ParameterSymbol && symbol.OwnerClass == nil {
		if owner := e.moduleForFile(symbol.Span.FileID); owner != nil {
			for _, candidate := range owner.Semantic.Symbols {
				if candidate != nil && candidate.Kind == semantic.MemberSymbol && candidate.Span == symbol.Span {
					return candidate
				}
			}
		}
	}
	return symbol
}
