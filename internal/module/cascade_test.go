package module

import (
	"strings"
	"testing"
)

// diagnosticCodes lists the codes a compilation reported, in order.
func diagnosticCodes(result CompilationResult) string {
	codes := make([]string, 0, len(result.Diagnostics))
	for _, item := range result.Diagnostics {
		codes = append(codes, item.Diagnostic.Code)
	}
	return strings.Join(codes, ",")
}

// A file that does not lex or parse reports that, and nothing that merely
// follows from it.
func TestLaterStagesDoNotEchoASyntaxError(t *testing.T) {
	for source, want := range map[string]string{
		// An unclosed string used to add "expected )" and a wrong argument count.
		"write(\"hello)\nwrite(\"b\")\n": "LEX007",
		// A statement from another language used to add four unknown names.
		"function f(a: Int) -> Int {\n    return a\n}\nwrite(f(2))\n": "PAR003",
		// A missing block brace used to add an unknown name.
		"if x > 3\n    write(1)\n": "PAR002",
		// A reserved keyword as a name is one diagnostic, however often it is used.
		"not: Int := 70\nnot = not + 5\nwrite(not)\n": "PAR015",
	} {
		result := compileWorkspace(map[string]string{"/app/main.ahd": source}, "/app/main.ahd")
		if got := diagnosticCodes(result); got != want {
			t.Errorf("%q\ncodes = %s, want %s\n%s", source, got, want, diagnosticsText(result))
		}
	}
}

// A syntax error before a lexical one is independent of it and is kept.
func TestSyntaxErrorBeforeALexicalErrorIsKept(t *testing.T) {
	result := compileWorkspace(map[string]string{"/app/main.ahd": "x := [1 2]\nwrite(\"hello)\n"}, "/app/main.ahd")
	if got := diagnosticCodes(result); got != "LEX007,PAR003" && got != "PAR003,LEX007" {
		t.Fatalf("codes = %s\n%s", got, diagnosticsText(result))
	}
}

// Semantic errors are still reported for a file that parses.
func TestSemanticErrorsAreReportedForAFileThatParses(t *testing.T) {
	result := compileWorkspace(map[string]string{"/app/main.ahd": "write(missing)\n"}, "/app/main.ahd")
	if got := diagnosticCodes(result); got != "SEM001" {
		t.Fatalf("codes = %s\n%s", got, diagnosticsText(result))
	}
}
