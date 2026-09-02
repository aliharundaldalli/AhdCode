package analysis

import (
	"sort"

	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
)

// References finds every use of the symbol at the given byte offset,
// scoped honestly to the current compile graph -- the clicked document plus
// everything it transitively imports -- rather than a full workspace-wide
// index, which no part of this tooling builds. When includeDeclaration is
// true, the declaration site itself is included alongside its uses.
//
// It reports nil when the cursor is not on a resolvable, non-builtin
// symbol.
func (store *Store) References(path string, offset int, includeDeclaration bool) []Location {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()

	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	node := findNodeAtOffset(entryModule.Parsed.Program, offset)
	if node == nil {
		return nil
	}
	clicked, ok := entryModule.Semantic.ResolvedSymbols[node]
	if !ok || clicked == nil || clicked.Builtin {
		return nil
	}
	declaration := cached.declarationSymbol(clicked)
	if declaration == nil {
		return nil
	}
	targets := cached.equivalentSymbols(declaration)
	isDeclarationOccurrence := declarationOccurrencePredicate(declaration)

	seen := make(map[Location]bool)
	var out []Location

	for _, candidateModule := range cached.modules {
		if candidateModule == nil {
			continue
		}
		modulePath, ok := cached.fileToPath[candidateModule.File.ID]
		if !ok {
			continue
		}
		for useNode, useSymbol := range candidateModule.Semantic.ResolvedSymbols {
			if !isAnyOf(useSymbol, targets) {
				continue
			}
			if isDeclarationOccurrence(useNode) {
				continue
			}
			location := Location{Path: modulePath, Span: useNode.Span()}
			if !seen[location] {
				seen[location] = true
				out = append(out, location)
			}
		}
	}
	if includeDeclaration {
		if declarationPath, ok := cached.fileToPath[declaration.Span.FileID]; ok {
			location := Location{Path: declarationPath, Span: declaration.Span}
			if !seen[location] {
				seen[location] = true
				out = append(out, location)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Span.Start.Offset < out[j].Span.Start.Offset
	})
	return out
}

// declarationOccurrencePredicate returns a function identifying which
// ResolvedSymbols node is "the declaration itself" for a given symbol, so
// the use-scanning loop can skip it (References reports it separately, once,
// as declaration.Span). A plain span comparison against declaration.Span is
// not enough: for a `name: Type := value` binding the analyzer records the
// SAME symbol pointer at two different nodes -- the whole *ast.VariableDecl
// statement, and its Target identifier expression alone -- so both must be
// recognized, or the narrower one leaks through as a spurious extra "use" at
// the declaration's own position.
func declarationOccurrencePredicate(declaration *semantic.Symbol) func(ast.Node) bool {
	declarationNode := declaration.Declaration
	var targetIdentifier ast.Node
	if variableDecl, ok := declarationNode.(*ast.VariableDecl); ok {
		if identifier, ok := variableDecl.Target.(*ast.IdentifierExpr); ok {
			targetIdentifier = identifier
		}
	}
	return func(node ast.Node) bool {
		if declarationNode != nil && node == declarationNode {
			return true
		}
		if targetIdentifier != nil && node == targetIdentifier {
			return true
		}
		return node.Span() == declaration.Span
	}
}

func isAnyOf(symbol *semantic.Symbol, targets []*semantic.Symbol) bool {
	for _, target := range targets {
		if symbol == target {
			return true
		}
	}
	return false
}

// equivalentSymbols lists every Symbol pointer that represents the same
// declared entity as declaration, across the whole compile graph: the
// declaration itself, plus -- when it is visible outside its own module --
// the single shared clone BuildModuleInterface produced for it, which every
// importer's own `Module.name` access or Class-member access resolves to
// verbatim (the same *ModuleInterface object is reused by every importer of
// one module within a single compile, so there is exactly one such clone to
// find, not one per importer).
func (e *entry) equivalentSymbols(declaration *semantic.Symbol) []*semantic.Symbol {
	targets := []*semantic.Symbol{declaration}
	owner := e.modules[module.ModuleID(declaration.OriginModuleID)]
	if owner == nil || owner.Interface == nil {
		return targets
	}
	if declaration.OwnerClass != nil {
		for _, class := range owner.Interface.Classes {
			if class != nil && class.Class == declaration.OwnerClass {
				if member := class.Members[declaration.Name]; member != nil {
					targets = append(targets, member)
				}
				break
			}
		}
		return targets
	}
	if exported := owner.Interface.Exports[declaration.Name]; exported != nil {
		targets = append(targets, exported)
	}
	return targets
}
