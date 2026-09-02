package lsp

import (
	"encoding/json"
	"testing"
)

func completionAt(t *testing.T, text, uri string, offset int) []lspCompletionItem {
	t.Helper()
	index := newLineIndex(text)
	position := index.OffsetToPosition(offset)

	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: text}})
	client.request(2, "textDocument/completion", textDocumentPositionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("completion returned an error: %+v", response.Error)
	}
	var items []lspCompletionItem
	if err := json.Unmarshal(response.Result, &items); err != nil {
		t.Fatalf("completion result = %s: %v", response.Result, err)
	}
	return items
}

func hasCompletionLabel(items []lspCompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func TestCompletionModuleNameAfterBring(t *testing.T) {
	text := "bring Ma\n"
	items := completionAt(t, text, "file:///main.ahd", len("bring Ma"))
	if !hasCompletionLabel(items, "Math") {
		t.Fatalf("expected Math among completions, got %#v", items)
	}
}

func TestCompletionNamespaceMember(t *testing.T) {
	text := "bring Math\nwrite(Math.P)\n"
	items := completionAt(t, text, "file:///main.ahd", len("bring Math\nwrite(Math.P"))
	if !hasCompletionLabel(items, "PI") {
		t.Fatalf("expected PI among completions, got %#v", items)
	}
}

// TestCompletionSQLiteArrivesThroughTheGenericModulePath is the v0.3.0
// acceptance check for the v0.2.2 architecture: the SQLite module reaches the
// protocol layer with no SQLite-specific code anywhere in internal/lsp.
func TestCompletionSQLiteArrivesThroughTheGenericModulePath(t *testing.T) {
	items := completionAt(t, "bring SQL\n", "file:///main.ahd", len("bring SQL"))
	if !hasCompletionLabel(items, "SQLite") {
		t.Fatalf("expected SQLite among module completions, got %#v", items)
	}
	text := "bring SQLite\nx := SQLite.\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	for _, label := range []string{"open", "nullValue", "fromInt", "fromReal", "fromString", "Database", "SQLiteValue", "SQLiteError"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("expected %s among SQLite member completions, got %#v", label, items)
		}
	}
	text = "from SQLite bring SQL\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	if !hasCompletionLabel(items, "SQLiteError") || !hasCompletionLabel(items, "SQLiteValue") {
		t.Fatalf("expected SQLiteError/SQLiteValue exports, got %#v", items)
	}
}

// TestCompletionHTTPAndHTMLArriveThroughTheGenericModulePath is the v0.4.0
// counterpart: HTTP and HTML reach completion with no module-specific code
// anywhere in internal/lsp.
func TestCompletionHTTPAndHTMLArriveThroughTheGenericModulePath(t *testing.T) {
	items := completionAt(t, "bring HT\n", "file:///main.ahd", len("bring HT"))
	if !hasCompletionLabel(items, "HTTP") || !hasCompletionLabel(items, "HTML") {
		t.Fatalf("expected HTTP and HTML among module completions, got %#v", items)
	}
	text := "bring HTTP\nx := HTTP.\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	for _, label := range []string{"server", "text", "html", "response", "redirect", "cookie", "deleteCookie", "sessions", "Server", "Request", "Response", "Cookie", "Session", "SessionStore", "HTTPError"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("expected %s among HTTP member completions, got %#v", label, items)
		}
	}
	text = "bring HTML\nx := HTML.\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	for _, label := range []string{"text", "element", "render", "document", "HTMLNode", "HTMLError"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("expected %s among HTML member completions, got %#v", label, items)
		}
	}
	text = "from HTTP bring HTTP\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	if !hasCompletionLabel(items, "HTTPError") {
		t.Fatalf("expected HTTPError export, got %#v", items)
	}
	text = "from HTTP bring Sess\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	if !hasCompletionLabel(items, "Session") || !hasCompletionLabel(items, "SessionStore") {
		t.Fatalf("expected Session/SessionStore exports, got %#v", items)
	}
	text = "from HTML bring HTML\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	if !hasCompletionLabel(items, "HTMLError") || !hasCompletionLabel(items, "HTMLNode") {
		t.Fatalf("expected HTMLError/HTMLNode exports, got %#v", items)
	}
}
