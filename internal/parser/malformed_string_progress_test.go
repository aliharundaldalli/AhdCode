package parser

import (
	"testing"
	"time"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/lexer"
	"ahdcode/internal/source"
)

const malformedStringProgressTimeout = 2 * time.Second

func parseWithHardTimeout(t *testing.T, text string) (lexerDiagnostics, parserDiagnostics []diagnostics.Diagnostic) {
	t.Helper()
	done := make(chan struct {
		lexed  lexer.Result
		parsed Result
	}, 1)
	go func() {
		file := source.NewFile(1, "test.ahd", text)
		lexed := lexer.Lex(file)
		parsed := Parse(file, lexed.Tokens)
		done <- struct {
			lexed  lexer.Result
			parsed Result
		}{lexed, parsed}
	}()
	select {
	case result := <-done:
		if result.parsed.Program == nil {
			t.Fatal("parser returned a nil program")
		}
		return result.lexed.Diagnostics, result.parsed.Diagnostics
	case <-time.After(malformedStringProgressTimeout):
		t.Fatalf("parser did not terminate within %s for %q", malformedStringProgressTimeout, text)
	}
	return nil, nil
}

func requireFrontendDiagnostic(t *testing.T, lexerDiagnostics, parserDiagnostics []diagnostics.Diagnostic) {
	t.Helper()
	if len(lexerDiagnostics) == 0 && len(parserDiagnostics) == 0 {
		t.Fatal("expected a diagnostic, got none")
	}
}

func TestMalformedStringAtModuleRootReportsDiagnostic(t *testing.T) {
	lexerDiagnostics, parserDiagnostics := parseWithHardTimeout(t, "\"{\\\"a\\\": 1}\"\n")
	requireFrontendDiagnostic(t, lexerDiagnostics, parserDiagnostics)
}

func TestMalformedStringInsideCallTerminates(t *testing.T) {
	cases := []string{
		"write(\"{\\\"a\\\": 1}\")\n",
		"write(\"{foo(}\")\n",
		"write(value: \"{\\\"a\\\": 1}\")\n",
	}
	for _, text := range cases {
		t.Run(text, func(t *testing.T) {
			lexerDiagnostics, parserDiagnostics := parseWithHardTimeout(t, text)
			requireFrontendDiagnostic(t, lexerDiagnostics, parserDiagnostics)
		})
	}
}

func TestMalformedStringNestedInListPairCallAndReturnTerminates(t *testing.T) {
	cases := []string{
		"[\"{\\\"a\\\": 1}\"]\n",
		"{\"k\": \"{\\\"a\\\": 1}\"}\n",
		"foo([\"{\\\"a\\\": 1}\"])\n",
		"foo({\"k\": \"{\\\"a\\\": 1}\"})\n",
		"foo(bar: \"{\\\"a\\\": 1}\")\n",
		"f: Function := () -> String {\n    return \"{\\\"a\\\": 1}\"\n}\n",
		"f: Function := () -> String {\n    return write(\"{\\\"a\\\": 1}\")\n}\n",
		"f: Function := () -> Nothing {\n    write(\"{foo(}\")\n}\n",
	}
	for _, text := range cases {
		t.Run(text, func(t *testing.T) {
			lexerDiagnostics, parserDiagnostics := parseWithHardTimeout(t, text)
			requireFrontendDiagnostic(t, lexerDiagnostics, parserDiagnostics)
		})
	}
}

func TestValidOrdinaryStringEscapesRemainUnchanged(t *testing.T) {
	result := parseText(t, "value := \"line\\n tab\\t quote\\\" tick\\' open\\{ close\\} slash\\\\\"\n")
	requireClean(t, result)
}

func TestValidRawStringsRemainUnchanged(t *testing.T) {
	result := parseText(t, "json := r'{\"a\": 1}'\npath := r\"C:\\new\\test\"\nquoted := r'foo \" bar \\\\ baz'\n")
	requireClean(t, result)
}
