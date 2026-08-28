package parser

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

// Result keeps both the typed AST and original token/trivia stream.
type Result struct {
	Program     *ast.Program
	Tokens      []token.Token
	Diagnostics []diagnostics.Diagnostic
}

func (r Result) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
