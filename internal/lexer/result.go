package lexer

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/syntax/token"
)

// Result is the complete lexical output for one source file.
type Result struct {
	Tokens      []token.Token
	Diagnostics []diagnostics.Diagnostic
}

// HasErrors reports whether lexing produced an error.
func (r Result) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
