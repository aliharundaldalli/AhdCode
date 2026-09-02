package analysis

import (
	"path/filepath"
	"testing"

	"ahdcode/internal/semantic"
)

func TestDocumentSymbolsListsTopLevelDeclarations(t *testing.T) {
	text := "score: Real := 85.0\n" +
		"limit: Constant Int := 10\n" +
		"square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	symbols := store.DocumentSymbols(path)
	if len(symbols) != 3 {
		t.Fatalf("expected 3 top-level symbols, got %d: %#v", len(symbols), symbols)
	}
	if symbols[0].Name != "score" || symbols[0].Kind != semantic.BindingSymbol {
		t.Fatalf("symbols[0] = %#v", symbols[0])
	}
	if symbols[1].Name != "limit" || symbols[1].Kind != semantic.BindingSymbol {
		t.Fatalf("symbols[1] = %#v", symbols[1])
	}
	if symbols[2].Name != "square" || symbols[2].Kind != semantic.FunctionSymbol || symbols[2].Detail != "square: (value: Int) -> Int" {
		t.Fatalf("symbols[2] = %#v", symbols[2])
	}
}

func TestDocumentSymbolsIncludesClassMembers(t *testing.T) {
	text := "Student: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n\n" +
		"    describe: Function := (\n    ) -> String {\n        return attribute.name\n    }\n" +
		"}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	symbols := store.DocumentSymbols(path)
	if len(symbols) != 1 || symbols[0].Name != "Student" || symbols[0].Kind != semantic.ClassSymbol {
		t.Fatalf("expected one Class symbol Student, got %#v", symbols)
	}
	children := symbols[0].Children
	if len(children) != 2 {
		t.Fatalf("expected 2 Class members, got %d: %#v", len(children), children)
	}
	if children[0].Name != "name" || children[0].Kind != semantic.MemberSymbol {
		t.Fatalf("children[0] = %#v", children[0])
	}
	if children[1].Name != "describe" || children[1].Kind != semantic.FunctionSymbol {
		t.Fatalf("children[1] = %#v", children[1])
	}
}

func TestDocumentSymbolsOnUnanalyzedDocumentReportsNil(t *testing.T) {
	store := NewStore()
	if symbols := store.DocumentSymbols("/nonexistent/main.ahd"); symbols != nil {
		t.Fatalf("expected nil symbols for an unopened document, got %#v", symbols)
	}
}
