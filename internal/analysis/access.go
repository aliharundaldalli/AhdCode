package analysis

import (
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
)

// enclosingClassAt returns the Class symbol for the innermost enclosing Class
// callable that contains offset, or nil when the cursor is in module scope or
// a non-class callable. This is a structural walk over the same ancestor path
// completion and signature help already use — not a second scope engine.
func enclosingClassAt(entryModule *module.Module, ancestors []ast.Node) *semantic.Symbol {
	for index := len(ancestors) - 1; index >= 0; index-- {
		switch ancestors[index].(type) {
		case *ast.FunctionDecl, *ast.StructureDecl:
			for outer := index - 1; outer >= 0; outer-- {
				if classDecl, ok := ancestors[outer].(*ast.ClassDecl); ok {
					if symbol, ok := entryModule.Semantic.ResolvedSymbols[classDecl]; ok {
						return symbol
					}
					return nil
				}
			}
			return nil
		}
	}
	return nil
}
