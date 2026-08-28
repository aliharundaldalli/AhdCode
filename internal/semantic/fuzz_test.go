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
		"use: Function := (operation: Function, x: Int) -> Int {\nreturn operation(x)\n}",
		"f: Function := (x: Int) -> Int { return x }\nf: Overload Function := (x: Real) -> Real { return x }\nf(1)",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		file := source.NewFile(1, "fuzz.ahd", text)
		parsed := parser.Parse(file, lexer.Lex(file).Tokens)
		result := Analyze(parsed)
		if result.ExpressionTypes == nil || result.NullStates == nil || result.ResolvedSymbols == nil || result.SelectedCallables == nil || result.SelectedFunctionValues == nil || result.OverloadResolutions == nil {
			t.Fatal("semantic side tables must always be initialized")
		}
	})
}
