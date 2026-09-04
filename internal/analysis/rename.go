package analysis

import (
	"sort"

	"ahdcode/internal/lexer"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

// TextEdit replaces one source span with new text.
type TextEdit struct {
	Path    string
	Span    source.Span
	NewText string
}

// PrepareRename reports whether the symbol at offset can be renamed and, when
// ok, the exact span of its name token.
func (store *Store) PrepareRename(path string, offset int) (source.Span, bool) {
	symbol, span, ok := store.renameTarget(path, offset)
	if !ok || symbol == nil {
		return source.Span{}, false
	}
	_ = symbol
	return span, true
}

// Rename produces text edits renaming the symbol at offset to newName across
// the current compile graph. newName must satisfy lexer.ValidIdentifier and
// must not be a reserved keyword.
func (store *Store) Rename(path string, offset int, newName string) ([]TextEdit, bool) {
	if !lexer.ValidIdentifier(newName) {
		return nil, false
	}
	if _, isKeyword := token.LookupKeyword(newName); isKeyword {
		return nil, false
	}
	symbol, declarationSpan, ok := store.renameTarget(path, offset)
	if !ok || symbol == nil {
		return nil, false
	}
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	if cached == nil {
		return nil, false
	}
	declaration := cached.declarationSymbol(symbol)
	if declaration == nil {
		return nil, false
	}
	targets := cached.equivalentSymbols(declaration)
	isDeclarationOccurrence := declarationOccurrencePredicate(declaration)

	type occurrence struct {
		path string
		span source.Span
	}
	seen := make(map[occurrence]bool)
	var edits []TextEdit

	addEdit := func(modulePath string, node ast.Node) {
		nameSpan := renameNameSpan(node, declarationSpan)
		if nameSpan.Empty() {
			return
		}
		key := occurrence{path: modulePath, span: nameSpan}
		if seen[key] {
			return
		}
		seen[key] = true
		edits = append(edits, TextEdit{Path: modulePath, Span: nameSpan, NewText: newName})
	}

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
				addEdit(modulePath, useNode)
				continue
			}
			if identifier, ok := useNode.(*ast.IdentifierExpr); ok {
				addEdit(modulePath, identifier)
			} else if member, ok := useNode.(*ast.MemberExpr); ok && member.Name == declaration.Name {
				addEdit(modulePath, member)
			}
		}
	}
	if len(edits) == 0 {
		addEdit(cached.fileToPath[declaration.Span.FileID], nil)
		if declarationNode := declaration.Declaration; declarationNode != nil {
			if variableDecl, ok := declarationNode.(*ast.VariableDecl); ok {
				if identifier, ok := variableDecl.Target.(*ast.IdentifierExpr); ok {
					addEdit(cached.fileToPath[declaration.Span.FileID], identifier)
				}
			}
		}
	}
	if len(edits) == 0 {
		return nil, false
	}
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].Path != edits[j].Path {
			return edits[i].Path < edits[j].Path
		}
		return edits[i].Span.Start.Offset < edits[j].Span.Start.Offset
	})
	return edits, true
}

func (store *Store) renameTarget(path string, offset int) (*semantic.Symbol, source.Span, bool) {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil, source.Span{}, false
	}
	node := findNodeAtOffset(entryModule.Parsed.Program, offset, cached.fileIDFor(canonical))
	if node == nil {
		return nil, source.Span{}, false
	}
	switch typed := node.(type) {
	case *ast.IdentifierExpr:
		symbol, ok := entryModule.Semantic.ResolvedSymbols[typed]
		if !ok || symbol == nil || symbol.Builtin {
			return nil, source.Span{}, false
		}
		return symbol, typed.Span(), true
	case *ast.MemberExpr:
		symbol, ok := entryModule.Semantic.ResolvedSymbols[typed]
		if ok && symbol != nil && !symbol.Builtin {
			return symbol, memberNameSpan(typed), true
		}
		return nil, source.Span{}, false
	default:
		if symbol, ok := entryModule.Semantic.ResolvedSymbols[node]; ok && symbol != nil && !symbol.Builtin {
			return symbol, renameNameSpan(node, symbol.Span), true
		}
	}
	return nil, source.Span{}, false
}

func renameNameSpan(node ast.Node, fallback source.Span) source.Span {
	switch typed := node.(type) {
	case *ast.IdentifierExpr:
		return typed.Span()
	case *ast.MemberExpr:
		return memberNameSpan(typed)
	case *ast.VariableDecl:
		if identifier, ok := typed.Target.(*ast.IdentifierExpr); ok {
			return identifier.Span()
		}
	}
	return fallback
}

func memberNameSpan(member *ast.MemberExpr) source.Span {
	span := member.Span()
	nameLen := len(member.Name)
	if nameLen == 0 {
		return span
	}
	end := span.End.Offset
	start := end - nameLen
	if start < span.Start.Offset {
		start = span.Start.Offset
	}
	return source.Span{FileID: span.FileID, Start: source.Position{Offset: start}, End: source.Position{Offset: end}}
}
