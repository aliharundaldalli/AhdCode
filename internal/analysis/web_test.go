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

func TestWebV016CompletionHoverAndSignatures(t *testing.T) {
	for _, tc := range []struct {
		text, marker string
		labels       []string
	}{
		{"bring Web\nWeb.\n", "Web.", []string{"context", "form", "errors", "RequestContext", "ValidationErrors", "OldInput", "routes", "RouteSet", "RouteGroup"}},
		{"bring Web\nfrom Web bring \n", "from Web bring ", []string{"RequestContext", "Form", "FormValueError", "WebContextError", "RouteSet", "RouteGroup", "WebRouteError"}},
		{"bring Web\nWeb.UI.\n", "Web.UI.", []string{"csrfField", "input", "form", "label", "select", "option", "button"}},
	} {
		path := filepath.Join(t.TempDir(), "main.ahd")
		store := NewStore()
		store.Open(path, tc.text)
		items := store.Completion(path, strings.Index(tc.text, tc.marker)+len(tc.marker))
		for _, label := range tc.labels {
			if !hasLabel(items, label) {
				t.Errorf("%s: missing %s in %#v", tc.marker, label, items)
			}
		}
	}
	text := `bring Web
from Web bring (Request, Response, RequestContext, SessionStore)
handle: Function := (request: Request, sessions: SessionStore) -> Response {
    context: Local RequestContext := Web.context(request, sessions)
    Web.UI.csrfField(context)
    return context.respond(Web.text("ok"))
}
`
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	for _, tc := range []struct{ call, result string }{
		{"Web.context(", "RequestContext"},
		{"Web.UI.csrfField(", "HTMLNode"},
		{"context.respond(", "Response"},
	} {
		at := strings.Index(text, tc.call)
		help, ok := store.SignatureHelp(path, at+len(tc.call))
		if !ok || !strings.Contains(help.Label, tc.result) {
			t.Errorf("signature %s: %#v, %v", tc.call, help, ok)
		}
		hover, ok := store.Hover(path, at+len(tc.call)-2)
		if !ok || !strings.Contains(hover.Text, tc.result) {
			t.Errorf("hover %s: %#v, %v", tc.call, hover, ok)
		}
	}
}
