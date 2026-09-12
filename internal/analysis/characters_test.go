package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// Characters reaches editors through the compiler's module interface, like
// every other standard module; nothing here is a hand-written table.

func TestCompletionOffersCharactersModuleName(t *testing.T) {
	text := "bring Cha\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	if items := store.Completion(path, len("bring Cha")); !hasLabel(items, "Characters") {
		t.Fatalf("expected Characters among module completions, got %#v", items)
	}
}

func TestCompletionOffersCharactersMembers(t *testing.T) {
	text := "bring Characters\nvalue := Characters.\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	items := store.Completion(path, len("bring Characters\nvalue := Characters."))
	for _, name := range []string{
		"list", "count", "codePoint", "fromCodePoint",
		"isLetter", "isDigit", "isWhitespace", "isUpper", "isLower",
		"isAlphaNumeric", "isPunctuation", "isSymbol",
	} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Characters member completions, got %#v", name, items)
		}
	}
}

func TestCompletionOffersCharactersErrorAfterFromBring(t *testing.T) {
	text := "from Characters bring \n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	if items := store.Completion(path, len("from Characters bring ")); !hasLabel(items, "CharactersError") {
		t.Fatalf("expected CharactersError among export completions, got %#v", items)
	}
}

func TestHoverShowsCharactersSignature(t *testing.T) {
	text := "bring Characters\nvalue: Int := Characters.codePoint(\"A\")\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	hover, ok := store.Hover(path, offsetOf(t, text, "codePoint")+1)
	if !ok {
		t.Fatal("no hover for Characters.codePoint")
	}
	if !strings.Contains(hover.Text, "codePoint") || !strings.Contains(hover.Text, "character: String") || !strings.Contains(hover.Text, "Int") {
		t.Fatalf("hover does not describe codePoint's signature: %q", hover.Text)
	}
}
