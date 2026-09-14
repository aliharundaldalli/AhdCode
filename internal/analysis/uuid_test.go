package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// UUID reaches editors through the compiler's module interface, like every
// other standard module.

func TestCompletionOffersUUIDModuleAndMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring \n")
	if items := store.Completion(path, len("bring ")); !hasLabel(items, "UUID") {
		t.Fatalf("expected UUID among module completions, got %#v", items)
	}
	text := "bring UUID\nvalue := UUID.\n"
	store.Open(path, text)
	items := store.Completion(path, len("bring UUID\nvalue := UUID."))
	for _, name := range []string{"v4", "v7", "parse", "isValid", "zero"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among UUID completions, got %#v", name, items)
		}
	}
	store.Open(path, "from UUID bring \n")
	items = store.Completion(path, len("from UUID bring "))
	for _, name := range []string{"UUIDValue", "UUIDError"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among UUID exports, got %#v", name, items)
		}
	}
}

func TestHoverShowsUUIDParseSignature(t *testing.T) {
	text := "bring UUID\nid := UUID.parse(\"x\")\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	hover, ok := store.Hover(path, offsetOf(t, text, "parse")+1)
	if !ok {
		t.Fatal("no hover for UUID.parse")
	}
	if !strings.Contains(hover.Text, "text: String") || !strings.Contains(hover.Text, "UUIDValue") {
		t.Fatalf("hover does not describe UUID.parse: %q", hover.Text)
	}
}
