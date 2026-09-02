package analysis

import (
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

// DocumentSymbol is one entry in a document's outline: a top-level
// declaration, or (for a Class) one of its members. Detail reuses the
// compiler's own hover rendering verbatim, so an outline entry's signature
// or type text never diverges from what Hover already shows for the same
// declaration.
type DocumentSymbol struct {
	Name     string
	Kind     semantic.SymbolKind
	Detail   string
	Span     source.Span
	Children []DocumentSymbol
}

// DocumentSymbols lists the outline of the most recently analyzed snapshot
// of path: every top-level declaration, and every method and attribute of
// every top-level Class. It reports nil for an unanalyzed document.
func (store *Store) DocumentSymbols(path string) []DocumentSymbol {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()

	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	resolved := entryModule.Semantic.ResolvedSymbols
	var out []DocumentSymbol
	for _, statement := range entryModule.Parsed.Program.Statements {
		symbol, ok := resolved[statement]
		if !ok || symbol == nil {
			continue
		}
		item := DocumentSymbol{Name: symbol.Name, Kind: symbol.Kind, Detail: renderHover(symbol), Span: statement.Span()}
		if class, ok := statement.(*ast.ClassDecl); ok {
			item.Children = classMemberSymbols(class, symbol, resolved)
		}
		out = append(out, item)
	}
	return out
}

// classMemberSymbols lists one Class's methods and attributes, in source
// order. Attributes are looked up through the Class's own Members map (the
// same map Definition already relies on for cross-module Class-member
// lookups) rather than the structure's constructor-parameter symbols,
// because Members carries the MemberSymbol identity a caller actually wants
// in an outline -- the constructor-parameter symbol sharing the same
// syntactic position exists only for analyzing the constructor body.
func classMemberSymbols(declaration *ast.ClassDecl, class *semantic.Symbol, resolved map[ast.Node]*semantic.Symbol) []DocumentSymbol {
	var out []DocumentSymbol
	for _, member := range declaration.Members {
		switch node := member.(type) {
		case *ast.FunctionDecl:
			symbol, ok := resolved[node]
			if !ok || symbol == nil {
				continue
			}
			out = append(out, DocumentSymbol{Name: symbol.Name, Kind: symbol.Kind, Detail: renderHover(symbol), Span: node.Span()})
		case *ast.StructureDecl:
			for index := range node.Parameters {
				parameter := &node.Parameters[index]
				attribute := class.Members[parameter.Name]
				if attribute == nil {
					continue
				}
				out = append(out, DocumentSymbol{Name: attribute.Name, Kind: attribute.Kind, Detail: renderHover(attribute), Span: parameter.Span()})
			}
			if node.Body != nil {
				for _, statement := range node.Body.Statements {
					declaration, ok := statement.(*ast.VariableDecl)
					if !ok {
						continue
					}
					symbol, ok := resolved[declaration]
					if !ok || symbol == nil {
						continue
					}
					out = append(out, DocumentSymbol{Name: symbol.Name, Kind: symbol.Kind, Detail: renderHover(symbol), Span: declaration.Span()})
				}
			}
		}
	}
	return out
}
