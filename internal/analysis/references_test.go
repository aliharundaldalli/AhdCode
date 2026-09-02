package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func offsetsIn(locations []Location, path string) []int {
	var out []int
	for _, location := range locations {
		if location.Path == path {
			out = append(out, location.Span.Start.Offset)
		}
	}
	sort.Ints(out)
	return out
}

func TestReferencesWithinASingleDocument(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\nwrite(score + 1.0)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	uses := store.References(path, offsetOf(t, text, "write(score)")+len("write(")+1, false)
	canonical := canonicalPath(path)
	got := offsetsIn(uses, canonical)
	want := []int{offsetOf(t, text, "score)"), offsetOf(t, text, "score + 1.0")}
	sort.Ints(want)
	if len(got) != len(want) {
		t.Fatalf("uses (declaration excluded) = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("uses (declaration excluded) = %v, want %v", got, want)
		}
	}

	withDeclaration := store.References(path, offsetOf(t, text, "score:")+1, true)
	gotWith := offsetsIn(withDeclaration, canonical)
	declarationOffset := offsetOf(t, text, "score:")
	found := false
	for _, offset := range gotWith {
		if offset == declarationOffset {
			found = true
		}
	}
	if !found || len(gotWith) != 3 {
		t.Fatalf("includeDeclaration=true offsets = %v, expected 3 including the declaration at %d", gotWith, declarationOffset)
	}
}

func TestReferencesCrossesIntoAnImportedModule(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	helperText := "greeting: String := \"hello\"\nwrite(greeting)\n"
	if err := os.WriteFile(helperPath, []byte(helperText), 0o600); err != nil {
		t.Fatal(err)
	}
	entryText := "bring Helper\nwrite(Helper.greeting)\nwrite(Helper.greeting)\n"

	store := NewStore()
	store.Open(entryPath, entryText)

	locations := store.References(entryPath, offsetOf(t, entryText, "Helper.greeting)")+len("Helper.")+1, true)
	if len(locations) != 4 {
		t.Fatalf("expected 4 locations (1 declaration + 1 use in Helper.ahd + 2 uses in main.ahd), got %d: %#v", len(locations), locations)
	}

	helperUses := offsetsIn(locations, canonicalPath(helperPath))
	if len(helperUses) != 2 {
		t.Fatalf("expected 2 locations in Helper.ahd (its own declaration + its own use), got %v", helperUses)
	}
	mainUses := offsetsIn(locations, canonicalPath(entryPath))
	if len(mainUses) != 2 {
		t.Fatalf("expected 2 locations in main.ahd, got %v", mainUses)
	}
}

func TestReferencesCrossesIntoAnImportedModuleForAClassMember(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	helperText := "Student: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n\n" +
		"    describe: Function := (\n    ) -> String {\n        return attribute.name\n    }\n" +
		"}\n"
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

	locations := store.References(entryPath, offsetOf(t, entryText, "student.name")+len("student.")+1, true)
	helperUses := offsetsIn(locations, canonicalPath(helperPath))
	if len(helperUses) != 2 {
		t.Fatalf("expected the declaration and the attribute.name use inside Helper.ahd, got %v (%#v)", helperUses, locations)
	}
	mainUses := offsetsIn(locations, canonicalPath(entryPath))
	if len(mainUses) != 1 {
		t.Fatalf("expected the student.name use in main.ahd, got %v", mainUses)
	}
}

func TestReferencesOnBuiltinReportsNil(t *testing.T) {
	text := "write(\"hi\")\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if locations := store.References(path, offsetOf(t, text, "write")+1, true); locations != nil {
		t.Fatalf("expected nil references for a builtin, got %#v", locations)
	}
}
