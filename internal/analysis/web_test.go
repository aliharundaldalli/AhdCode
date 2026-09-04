package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// The language server learns about Web the same way it learns about a module
// the user wrote: by compiling it and reading the interface the compiler
// builds. Nothing in internal/lsp or internal/analysis holds a catalog of
// Web's API, so these tests assert discoverability, not a hardcoded list.

func TestCompletionOffersBundledWebModuleName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring We\n")

	items := store.Completion(path, len("bring We"))
	if !hasLabel(items, "Web") {
		t.Fatalf("expected Web among module completions, got %#v", items)
	}
}

// Web's exports come from the compiled interface, so the facade's re-exported
// types are offered alongside its own functions.
func TestCompletionOffersWebExportsFromTheCompiledInterface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring Web\nfrom Web bring R\n")

	items := store.Completion(path, len("bring Web\nfrom Web bring R"))
	for _, expected := range []string{"Request", "Response"} {
		if !hasLabel(items, expected) {
			t.Errorf("expected %s among Web export completions, got %#v", expected, items)
		}
	}
}

// A namespace member completion after `Web.` reads the same interface.
func TestCompletionOffersWebNamespaceMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	text := "bring Web\nvalue: String := Web.re\n"
	store.Open(path, text)

	items := store.Completion(path, strings.Index(text, "Web.re")+len("Web.re"))
	if !hasLabel(items, "redirect") && !hasLabel(items, "render") && !hasLabel(items, "response") {
		t.Fatalf("expected Web namespace members after \"Web.re\", got %#v", items)
	}
}

// Hover on a Web export reports the signature the compiler derived from the
// framework's own AhdCode source.
func TestHoverDescribesWebExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	text := "bring Web\nfrom Web bring Response\n"
	store.Open(path, text)

	hover, ok := store.Hover(path, strings.Index(text, "from Web bring Response")+len("from Web bring R"))
	if !ok {
		t.Fatal("expected hover content for a Web export")
	}
	if !strings.Contains(hover.Text, "Response") {
		t.Errorf("hover did not mention Response: %q", hover.Text)
	}
}
