package analysis

import (
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// InlayHintKind distinguishes type hints from parameter-name hints.
type InlayHintKind int

const (
	InlayHintType InlayHintKind = iota
	InlayHintParameter
)

// InlayHint is one inlay hint at a source position.
type InlayHint struct {
	Offset  int
	Length  int
	Label   string
	Kind    InlayHintKind
	Padding bool
}

// InlayHints returns restrained type and parameter-name hints for path.
func (store *Store) InlayHints(path string) []InlayHint {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	var hints []InlayHint
	resolved := entryModule.Semantic.ResolvedSymbols
	for _, statement := range entryModule.Parsed.Program.Statements {
		collectInlayStmt(statement, entryModule, resolved, &hints)
	}
	return hints
}

func collectInlayStmt(statement ast.Stmt, entryModule *module.Module, resolved map[ast.Node]*semantic.Symbol, hints *[]InlayHint) {
	switch node := statement.(type) {
	case *ast.VariableDecl:
		if node.Inferred && node.Type == nil {
			if symbol, ok := resolved[node]; ok && symbol != nil && !types.IsInvalid(symbol.Type) {
				*hints = append(*hints, InlayHint{
					Offset:  nameOffset(node),
					Length:  0,
					Label:   ": " + types.Display(symbol.Type),
					Kind:    InlayHintType,
					Padding: true,
				})
			}
		}
		collectInlayBlock(node.Initializer, entryModule, resolved, hints)
	case *ast.FunctionDecl:
		if node.Body != nil {
			for _, inner := range node.Body.Statements {
				collectInlayStmt(inner, entryModule, resolved, hints)
			}
		}
	case *ast.ClassDecl:
		for _, member := range node.Members {
			collectInlayStmt(member, entryModule, resolved, hints)
		}
	}
	collectInlayExpr(stmtExpr(statement), entryModule, resolved, hints)
}

func stmtExpr(statement ast.Stmt) ast.Expr {
	switch node := statement.(type) {
	case *ast.ExprStmt:
		return node.Expression
	default:
		return nil
	}
}

func collectInlayBlock(expression ast.Expr, entryModule *module.Module, resolved map[ast.Node]*semantic.Symbol, hints *[]InlayHint) {
	if expression == nil {
		return
	}
	collectInlayExpr(expression, entryModule, resolved, hints)
}

func collectInlayExpr(expression ast.Expr, entryModule *module.Module, resolved map[ast.Node]*semantic.Symbol, hints *[]InlayHint) {
	if expression == nil {
		return
	}
	switch node := expression.(type) {
	case *ast.CallExpr:
		callable, ok := entryModule.Semantic.SelectedCallables[node]
		if ok && callable != nil && callable.Signature != nil {
			for index := range node.Arguments {
				argument := &node.Arguments[index]
				if argument.Name != "" {
					continue
				}
				if index >= len(callable.Signature.Parameters) {
					break
				}
				paramName := callable.Signature.Parameters[index].Name
				if paramName == "" {
					continue
				}
				if identifier, ok := argument.Value.(*ast.IdentifierExpr); ok && identifier.Name == paramName {
					continue
				}
				*hints = append(*hints, InlayHint{
					Offset:  argument.Span().Start.Offset,
					Length:  0,
					Label:   paramName + ": ",
					Kind:    InlayHintParameter,
					Padding: false,
				})
			}
		}
		for index := range node.Arguments {
			collectInlayExpr(node.Arguments[index].Value, entryModule, resolved, hints)
		}
		collectInlayExpr(node.Callee, entryModule, resolved, hints)
	default:
		walkExprChildren(expression, entryModule, resolved, hints)
	}
}

func walkExprChildren(expression ast.Expr, entryModule *module.Module, resolved map[ast.Node]*semantic.Symbol, hints *[]InlayHint) {
	for _, child := range children(expression) {
		if expr, ok := child.(ast.Expr); ok {
			collectInlayExpr(expr, entryModule, resolved, hints)
		} else if block, ok := child.(*ast.Block); ok {
			for _, statement := range block.Statements {
				collectInlayStmt(statement, entryModule, resolved, hints)
			}
		}
	}
}

func nameOffset(decl *ast.VariableDecl) int {
	if identifier, ok := decl.Target.(*ast.IdentifierExpr); ok {
		return identifier.Span().End.Offset
	}
	return decl.Span().Start.Offset
}

// HintPosition converts a hint offset to a source span for LSP placement.
func HintPosition(text string, hint InlayHint) source.Span {
	offset := hint.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	return source.Span{
		Start: source.Position{Offset: offset},
		End:   source.Position{Offset: offset},
	}
}
