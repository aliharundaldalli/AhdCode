package parser

import (
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/source"
)

func FuzzParserNeverPanics(f *testing.F) {
	seeds := []string{
		"", "2^3^2", "swap(a b)", "Student(name: \"Ali\")",
		"if true {\nreturn\n}", "state x { condition default {} }",
		"name: Pair<String, Int> := null", "\xff\xfe", `"x={call(1)}"`,
		`write("{foo(}")`, `write("{\"a\": 1}")`,
		"f: Function := () -> String {\n    return write(\"{\\\"a\\\": 1}\")\n}\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		file := source.NewFile(1, "fuzz.ahd", text)
		lexed := lexer.Lex(file)
		result := Parse(file, lexed.Tokens)
		if result.Program == nil || len(result.Tokens) == 0 {
			t.Fatal("parser must always return a program and token stream")
		}
		span := result.Program.Span()
		if span.Start.Offset < 0 || span.End.Offset < span.Start.Offset || span.End.Offset > len(text) {
			t.Fatalf("invalid program span: %+v for %d bytes", span, len(text))
		}
	})
}
