package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// offsetOf returns the byte offset of the first occurrence of needle in
// text, or fails the test.
func offsetOf(t *testing.T, text, needle string) int {
	t.Helper()
	index := strings.Index(text, needle)
	if index < 0 {
		t.Fatalf("needle %q not found in %q", needle, text)
	}
	return index
}

func TestHoverVariableDeclarationAndUse(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	declaration, ok := store.Hover(path, offsetOf(t, text, "score")+1)
	if !ok || declaration.Text != "score: Real" {
		t.Fatalf("declaration hover = %#v, ok = %v", declaration, ok)
	}

	use, ok := store.Hover(path, offsetOf(t, text, "write(score")+len("write(")+1)
	if !ok || use.Text != "score: Real" {
		t.Fatalf("use hover = %#v, ok = %v", use, ok)
	}
}

func TestHoverConstantDeclaration(t *testing.T) {
	text := "limit: Constant Int := 10\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "limit")+1)
	if !ok || hover.Text != "Constant limit: Int" {
		t.Fatalf("hover = %#v, ok = %v", hover, ok)
	}
}

func TestHoverFunctionDeclarationAndUse(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\nresult := square(5)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	declaration, ok := store.Hover(path, offsetOf(t, text, "square:")+1)
	if !ok || declaration.Text != "square: (value: Int) -> Int" {
		t.Fatalf("function declaration hover = %#v, ok = %v", declaration, ok)
	}

	use, ok := store.Hover(path, offsetOf(t, text, "square(5)")+1)
	if !ok || use.Text != "square: (value: Int) -> Int" {
		t.Fatalf("function use hover = %#v, ok = %v", use, ok)
	}
}

func TestHoverFunctionParameterDeclarationAndUse(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	declaration, ok := store.Hover(path, offsetOf(t, text, "value: Int")+1)
	if !ok || declaration.Text != "value: Int" {
		t.Fatalf("parameter declaration hover = %#v, ok = %v", declaration, ok)
	}

	use, ok := store.Hover(path, offsetOf(t, text, "return value")+len("return ")+1)
	if !ok || use.Text != "value: Int" {
		t.Fatalf("parameter use hover = %#v, ok = %v", use, ok)
	}
}

func TestHoverClassDeclaration(t *testing.T) {
	text := "Student: Class<> := {\n    structure: Attributes := (\n        name: String\n    )\n}\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "Student:")+1)
	if !ok || hover.Text != "Class Student" {
		t.Fatalf("class hover = %#v, ok = %v", hover, ok)
	}
}

func TestHoverBuiltinStandardModuleMember(t *testing.T) {
	text := "bring Math\nwrite(Math.PI)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "Math.PI")+len("Math.")+1)
	if !ok || hover.Text != "Constant PI: Real" {
		t.Fatalf("Math.PI hover = %#v, ok = %v", hover, ok)
	}
}

func TestHoverNamespaceIdentifier(t *testing.T) {
	text := "bring Math\nwrite(Math.PI)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "Math.PI")+1)
	if !ok || hover.Text != "module Math" {
		t.Fatalf("Math namespace hover = %#v, ok = %v", hover, ok)
	}
}

func TestHoverPositionWithNoSymbolReturnsNoHover(t *testing.T) {
	text := "x: Int := 1 + 2\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	// The literal "1" and the "+" operator have no resolved symbol.
	_, ok := store.Hover(path, offsetOf(t, text, "1 + 2"))
	if ok {
		t.Fatal("expected no hover over a bare literal")
	}
}

func TestHoverOnUnanalyzedDocumentReturnsNoHover(t *testing.T) {
	store := NewStore()
	_, ok := store.Hover("/does/not/exist.ahd", 0)
	if ok {
		t.Fatal("expected no hover for a document that was never opened")
	}
}

func TestHoverUsesInMemoryTextAfterDidChange(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, "first: Int := 1\n")
	if _, ok := store.Hover(path, offsetOf(t, "first: Int := 1\n", "first")+1); !ok {
		t.Fatal("expected a hover on the initial document")
	}

	updated := "renamed: Real := 2.5\n"
	store.Change(path, updated)
	hover, ok := store.Hover(path, offsetOf(t, updated, "renamed")+1)
	if !ok || hover.Text != "renamed: Real" {
		t.Fatalf("expected hover to reflect the changed buffer, got %#v, ok=%v", hover, ok)
	}
	if _, ok := store.Hover(path, offsetOf(t, updated, "renamed")+1); !ok {
		t.Fatal("hover regressed after didChange")
	}
	// The old name no longer exists in the buffer at all, so searching for
	// it would be meaningless; instead confirm the old symbol name is gone
	// from hover text at the (now different) declaration position.
	if strings.Contains(hover.Text, "first") {
		t.Fatalf("hover still shows stale pre-change content: %#v", hover)
	}
}

func TestHoverUnicodeSource(t *testing.T) {
	text := "isim: String := \"Ayşe 🙂\"\nwrite(isim)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "isim")+1)
	if !ok || hover.Text != "isim: String" {
		t.Fatalf("hover with Unicode source = %#v, ok = %v", hover, ok)
	}

	use, ok := store.Hover(path, offsetOf(t, text, "write(isim")+len("write(")+1)
	if !ok || use.Text != "isim: String" {
		t.Fatalf("use hover with Unicode source = %#v, ok = %v", use, ok)
	}
}

func TestHoverHTTPCookieAndSessionArriveThroughTheModuleInterface(t *testing.T) {
	text := "bring HTTP\nfrom HTTP bring Session\ncookie := HTTP.cookie(\"a\", \"1\")\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "HTTP.cookie")+len("HTTP."))
	if !ok || !strings.Contains(hover.Text, "cookie") || !strings.Contains(hover.Text, "Cookie") {
		t.Fatalf("HTTP.cookie hover = %#v, ok = %v", hover, ok)
	}
}
