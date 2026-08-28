package semantic

import (
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
)

func FuzzSemanticNeverPanics(f *testing.F) {
	seeds := []string{
		"", "x: Int := 1", "x: Int := null\nx + 1", "if true {}",
		"value: Constant Int := 2 ^ 3", "Student(name: \"Ali\")", "\xff\xfe",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		file := source.NewFile(1, "fuzz.ahd", text)
		parsed := parser.Parse(file, lexer.Lex(file).Tokens)
		result := Analyze(parsed)
		if result.ExpressionTypes == nil || result.NullStates == nil || result.ResolvedSymbols == nil {
			t.Fatal("semantic side tables must always be initialized")
		}
	})
}
