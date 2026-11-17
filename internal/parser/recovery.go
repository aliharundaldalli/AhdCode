package parser

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/syntax/ast"
)

// One mistake should read as one diagnostic. Three rules keep a single
// syntax error from fanning out:
//
//   - a statement that reported its own error is skipped to the end of its
//     line silently (parseStatementList);
//   - a line keeps only its first syntax error (firstErrorPerLine), because
//     whatever follows on that line was parsed from an already broken state;
//   - a reserved keyword used as a name is reported once (reserved.go).
//
// Diagnostic codes and their meanings are unchanged; only repeats are removed.

// firstErrorPerLine keeps the first diagnostic that starts on each line.
func firstErrorPerLine(items []diagnostics.Diagnostic) []diagnostics.Diagnostic {
	seen := make(map[int]bool, len(items))
	kept := items[:0]
	for _, item := range items {
		line := item.Span.Start.Line
		if item.Severity == diagnostics.SeverityError {
			if seen[line] {
				continue
			}
			seen[line] = true
		}
		kept = append(kept, item)
	}
	return kept
}

// foreignStatementHints explain the statement keywords of other languages
// that AhdCode does not have, where "expected newline" alone says little.
var foreignStatementHints = map[string]string{
	"function": "AhdCode has no 'function' statement; declare a Function as name: Function := (parameter: Type) -> Type { ... }",
	"func":     "AhdCode has no 'func' statement; declare a Function as name: Function := (parameter: Type) -> Type { ... }",
	"fn":       "AhdCode has no 'fn' statement; declare a Function as name: Function := (parameter: Type) -> Type { ... }",
	"fun":      "AhdCode has no 'fun' statement; declare a Function as name: Function := (parameter: Type) -> Type { ... }",
	"def":      "AhdCode has no 'def' statement; declare a Function as name: Function := (parameter: Type) -> Type { ... }",
	"var":      "AhdCode has no 'var' statement; declare a name as name := value or name: Type := value",
	"let":      "AhdCode has no 'let' statement; declare a name as name := value or name: Type := value",
	"const":    "AhdCode has no 'const' statement; declare a constant as name: Constant Type := value",
	"class":    "AhdCode has no 'class' statement; declare a Class as Name: Class<> := { ... }",
	"import":   "AhdCode has no 'import' statement; use bring Module or from Module bring (name)",
	"print":    "AhdCode writes output with write(value)",
	"elif":     "AhdCode spells this branch else if",
	"switch":   "AhdCode spells this statement state value { condition ... { } }",
	"try":      "AhdCode spells this statement attempt { ... } except { ... }",
	"throw":    "AhdCode raises an error with toss",
	"raise":    "AhdCode raises an error with toss",
}

// foreignStatementHint recognizes a statement that is nothing but one of
// those words, which is how the parser reads `function name(...)`.
func foreignStatementHint(statement ast.Stmt) (string, bool) {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return "", false
	}
	identifier, ok := expression.Expression.(*ast.IdentifierExpr)
	if !ok {
		return "", false
	}
	hint, known := foreignStatementHints[identifier.Name]
	return hint, known
}
