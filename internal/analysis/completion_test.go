package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func hasLabel(items []CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func TestCompletionModuleNameAfterBring(t *testing.T) {
	text := "bring Ma\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("bring Ma"))
	if !hasLabel(items, "Math") {
		t.Fatalf("expected Math among module completions, got %#v", items)
	}
	if hasLabel(items, "JSON") {
		t.Fatalf("expected JSON to be filtered out by prefix \"Ma\", got %#v", items)
	}
}

func TestCompletionModuleNameAfterFrom(t *testing.T) {
	text := "from Ma\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("from Ma"))
	if !hasLabel(items, "Math") {
		t.Fatalf("expected Math among module completions, got %#v", items)
	}
}

func TestCompletionSiblingModuleName(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	if err := os.WriteFile(helperPath, []byte("x: Int := 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	text := "bring Hel\n"
	store := NewStore()
	store.Open(entryPath, text)

	items := store.Completion(entryPath, len("bring Hel"))
	if !hasLabel(items, "Helper") {
		t.Fatalf("expected Helper among module completions, got %#v", items)
	}
}

func TestCompletionExportNameAfterFromBring(t *testing.T) {
	text := "from Math bring P\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("from Math bring P"))
	if !hasLabel(items, "PI") {
		t.Fatalf("expected PI among export completions, got %#v", items)
	}
	if hasLabel(items, "sqrt") {
		t.Fatalf("expected sqrt to be filtered out by prefix \"P\", got %#v", items)
	}
}

func TestCompletionNamespaceMember(t *testing.T) {
	text := "bring Math\nwrite(Math.P)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, offsetOf(t, text, "Math.P")+len("Math.P"))
	if !hasLabel(items, "PI") {
		t.Fatalf("expected PI among Math member completions, got %#v", items)
	}
}

func TestCompletionClassMember(t *testing.T) {
	text := "Student: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n" +
		"}\n" +
		"s: Student := Student(name: \"Ada\")\n" +
		"write(s.n)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, offsetOf(t, text, "s.n")+len("s.n"))
	if !hasLabel(items, "name") {
		t.Fatalf("expected name among Class member completions, got %#v", items)
	}
}

func TestCompletionClassMemberIncludesInheritedMembers(t *testing.T) {
	text := "Person: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n" +
		"}\n" +
		"Student: Class<Person> := {\n" +
		"    structure: Attributes := (\n        SuperClass.attributes\n        number: Int\n    )\n" +
		"}\n" +
		"s: Student := Student(name: \"Ada\", number: 1)\n" +
		"write(s.n)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, offsetOf(t, text, "s.n")+len("s.n"))
	if !hasLabel(items, "name") || !hasLabel(items, "number") {
		t.Fatalf("expected both inherited name and own number among Class member completions, got %#v", items)
	}
}

// TestCompletionClassMemberExcludesConfidentialMembers is a regression test:
// Symbol.Members is the analyzer's own internal member-lookup table, which
// -- unlike a module's ExportNames -- carries every member regardless of
// visibility. Completion must never surface a Confidential member of an
// externally-referenced instance; a missing suggestion is acceptable, a
// wrong (inaccessible) one is not.
func TestCompletionClassMemberExcludesConfidentialMembers(t *testing.T) {
	classDecl := "Animal: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n\n" +
		"    speak: Function := (\n    ) -> String {\n        return \"...\"\n    }\n\n" +
		"    secret: Confidential Function := (\n    ) -> Int {\n        return 1\n    }\n" +
		"}\n" +
		"a: Animal := Animal(name: \"Rex\")\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()

	confidentialPrefixText := classDecl + "write(a.se)\n"
	store.Open(path, confidentialPrefixText)
	items := store.Completion(path, offsetOf(t, confidentialPrefixText, "a.se)")+len("a.se"))
	if hasLabel(items, "secret") {
		t.Fatalf("expected the Confidential member 'secret' to be excluded from completion, got %#v", items)
	}
	if hasLabel(items, "speak") {
		t.Fatalf("expected 'speak' to not match prefix 'se', got %#v", items)
	}

	publicPrefixText := classDecl + "write(a.sp)\n"
	store.Open(path, publicPrefixText)
	spItems := store.Completion(path, offsetOf(t, publicPrefixText, "a.sp)")+len("a.sp"))
	if !hasLabel(spItems, "speak") {
		t.Fatalf("expected the public member 'speak' to still be suggested, got %#v", spItems)
	}

	noPrefixText := classDecl + "write(a.)\n"
	store.Open(path, noPrefixText)
	allItems := store.Completion(path, offsetOf(t, noPrefixText, "a.)")+len("a."))
	if hasLabel(allItems, "secret") {
		t.Fatalf("expected 'secret' excluded even with no prefix filter at all, got %#v", allItems)
	}
	if !hasLabel(allItems, "speak") || !hasLabel(allItems, "name") {
		t.Fatalf("expected the public members 'speak' and 'name' with no prefix filter, got %#v", allItems)
	}
}

// TestCompletionClassMemberInheritedConfidentialAlsoExcluded proves the
// exclusion holds across an inheritance chain too, not only for a class's
// own directly declared Confidential members.
func TestCompletionClassMemberInheritedConfidentialAlsoExcluded(t *testing.T) {
	directory := t.TempDir()
	basePath := filepath.Join(directory, "Base.ahd")
	baseText := "Animal: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n\n" +
		"    secret: Confidential Function := (\n    ) -> Int {\n        return 1\n    }\n" +
		"}\n"
	if err := os.WriteFile(basePath, []byte(baseText), 0o600); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(directory, "main.ahd")
	entryText := "from Base bring Animal\n" +
		"Dog: Class<Animal> := {\n" +
		"    structure: Attributes := (\n        SuperClass.attributes\n    )\n\n" +
		"    bark: Function := (\n    ) -> String {\n        return \"Bark\"\n    }\n" +
		"}\n" +
		"d: Dog := Dog(name: \"Rex\")\n" +
		"write(d.se)\n"
	store := NewStore()
	store.Open(entryPath, entryText)

	items := store.Completion(entryPath, offsetOf(t, entryText, "d.se)")+len("d.se"))
	if hasLabel(items, "secret") {
		t.Fatalf("expected the inherited Confidential member 'secret' to be excluded, got %#v", items)
	}
}

// TestCompletionClassMemberWrongTypeReceiverGetsNothing proves a receiver
// with no Class type at all (a plain Int) never surfaces unrelated Class
// members -- there is nothing to prove from ExpressionTypes, so completion
// returns nothing rather than guessing.
func TestCompletionClassMemberWrongTypeReceiverGetsNothing(t *testing.T) {
	text := "Animal: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n" +
		"}\n" +
		"n: Int := 5\n" +
		"write(n.na)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, offsetOf(t, text, "n.na)")+len("n.na"))
	if len(items) != 0 {
		t.Fatalf("expected no completions for an Int receiver, got %#v", items)
	}
}

func TestCompletionLocalsParametersAndModuleRoot(t *testing.T) {
	text := "limit: Constant Int := 10\n" +
		"square: Function := (\n" +
		"    value: Int\n" +
		") -> Int {\n" +
		"    total: Int := value\n" +
		"    return to\n" +
		"}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, offsetOf(t, text, "return to")+len("return to"))
	if !hasLabel(items, "total") {
		t.Fatalf("expected local binding 'total' among completions, got %#v", items)
	}

	allItems := store.Completion(path, offsetOf(t, text, "return to"))
	if !hasLabel(allItems, "value") {
		t.Fatalf("expected parameter 'value' among completions, got %#v", allItems)
	}
	if !hasLabel(allItems, "limit") {
		t.Fatalf("expected module-root 'limit' among completions, got %#v", allItems)
	}
	if !hasLabel(allItems, "square") {
		t.Fatalf("expected module-root 'square' among completions, got %#v", allItems)
	}
	if !hasLabel(allItems, "if") {
		t.Fatalf("expected keyword 'if' among completions, got %#v", allItems)
	}
}

func TestCompletionOnUnopenedDocumentReportsNil(t *testing.T) {
	store := NewStore()
	if items := store.Completion("/nonexistent/main.ahd", 0); items != nil {
		t.Fatalf("expected nil completions for an unopened document, got %#v", items)
	}
}
