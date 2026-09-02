package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefinitionVariableUseJumpsToItsDeclaration(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	location, ok := store.Definition(path, offsetOf(t, text, "write(score")+len("write(")+1)
	if !ok {
		t.Fatal("expected a definition for the use of score")
	}
	if location.Path != canonicalPath(path) {
		t.Fatalf("definition path = %q, want %q", location.Path, canonicalPath(path))
	}
	declarationStart := offsetOf(t, text, "score: Real")
	if location.Span.Start.Offset != declarationStart {
		t.Fatalf("definition span start = %d, want %d", location.Span.Start.Offset, declarationStart)
	}
}

func TestDefinitionFunctionCallJumpsToItsDeclaration(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\nresult := square(5)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	location, ok := store.Definition(path, offsetOf(t, text, "square(5)")+1)
	if !ok {
		t.Fatal("expected a definition for the call to square")
	}
	declarationStart := offsetOf(t, text, "square: Function")
	if location.Span.Start.Offset != declarationStart {
		t.Fatalf("definition span start = %d, want %d", location.Span.Start.Offset, declarationStart)
	}
}

func TestDefinitionCrossesIntoAnImportedModule(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	helperText := "greeting: String := \"hello\"\n"
	if err := os.WriteFile(helperPath, []byte(helperText), 0o600); err != nil {
		t.Fatal(err)
	}
	entryText := "bring Helper\nwrite(Helper.greeting)\n"

	store := NewStore()
	store.Open(entryPath, entryText)

	location, ok := store.Definition(entryPath, offsetOf(t, entryText, "Helper.greeting")+len("Helper."))
	if !ok {
		t.Fatal("expected a definition crossing into Helper.ahd")
	}
	if location.Path != canonicalPath(helperPath) {
		t.Fatalf("definition path = %q, want Helper.ahd's own path %q", location.Path, canonicalPath(helperPath))
	}
	declarationStart := offsetOf(t, helperText, "greeting: String")
	if location.Span.Start.Offset != declarationStart {
		t.Fatalf("definition span start = %d, want %d", location.Span.Start.Offset, declarationStart)
	}
}

func TestDefinitionCrossesIntoAnImportedModuleForAClassMember(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	helperText := "Student: Class<> := {\n    structure: Attributes := (\n        name: String\n    )\n}\n"
	if err := os.WriteFile(helperPath, []byte(helperText), 0o600); err != nil {
		t.Fatal(err)
	}
	entryText := "from Helper bring Student\n" +
		"student: Student := Student(name: \"Ada\")\n" +
		"write(student.name)\n"

	store := NewStore()
	result := store.Open(entryPath, entryText)
	if diags := result.Diagnostics[canonicalPath(entryPath)]; len(diags) != 0 {
		t.Fatalf("expected the entry to be clean, got %v", diags)
	}

	location, ok := store.Definition(entryPath, offsetOf(t, entryText, "student.name")+len("student.")+1)
	if !ok {
		t.Fatal("expected a definition crossing into Helper.ahd for the Class member")
	}
	if location.Path != canonicalPath(helperPath) {
		t.Fatalf("definition path = %q, want Helper.ahd's own path %q", location.Path, canonicalPath(helperPath))
	}
	declarationStart := offsetOf(t, helperText, "name: String")
	if location.Span.Start.Offset != declarationStart {
		t.Fatalf("definition span start = %d, want %d", location.Span.Start.Offset, declarationStart)
	}
}

func TestDefinitionOnBuiltinReportsNoResult(t *testing.T) {
	text := "write(\"hi\")\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if _, ok := store.Definition(path, offsetOf(t, text, "write")+1); ok {
		t.Fatal("expected no definition for a builtin with no AhdCode-source declaration")
	}
}

func TestDefinitionWithNoResolvedSymbolReportsNoResult(t *testing.T) {
	text := "// just a comment\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if _, ok := store.Definition(path, 3); ok {
		t.Fatal("expected no definition inside a comment")
	}
}

func TestDefinitionOnUnopenedDocumentReportsNoResult(t *testing.T) {
	store := NewStore()
	if _, ok := store.Definition("/nonexistent/main.ahd", 0); ok {
		t.Fatal("expected no definition for a document that was never opened")
	}
}
