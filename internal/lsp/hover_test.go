package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

func hoverAt(t *testing.T, text, uri string, needle string, offsetWithinNeedle int) lspHover {
	t.Helper()
	index := newLineIndex(text)
	byteOffset := strings.Index(text, needle) + offsetWithinNeedle
	if byteOffset < 0 {
		t.Fatalf("needle %q not found in text", needle)
	}
	position := index.OffsetToPosition(byteOffset)

	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: text}})
	client.request(2, "textDocument/hover", textDocumentPositionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("hover returned an error: %+v", response.Error)
	}
	var hover lspHover
	if err := json.Unmarshal(response.Result, &hover); err != nil {
		t.Fatalf("hover result = %s: %v", response.Result, err)
	}
	return hover
}

func hoverIsNull(t *testing.T, text, uri, needle string, offsetWithinNeedle int) bool {
	t.Helper()
	index := newLineIndex(text)
	byteOffset := strings.Index(text, needle) + offsetWithinNeedle
	position := index.OffsetToPosition(byteOffset)

	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: text}})
	client.request(2, "textDocument/hover", textDocumentPositionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	return string(response.Result) == "null" || len(response.Result) == 0
}

func TestHoverVariableDeclaration(t *testing.T) {
	text := "score: Real := 85.0\n"
	hover := hoverAt(t, text, "file:///main.ahd", "score", 1)
	if hover.Contents.Value != "score: Real" {
		t.Fatalf("hover = %+v", hover)
	}
	if hover.Contents.Kind != "plaintext" {
		t.Fatalf("kind = %q", hover.Contents.Kind)
	}
}

func TestHoverVariableUse(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\n"
	hover := hoverAt(t, text, "file:///main.ahd", "write(score", len("write(")+1)
	if hover.Contents.Value != "score: Real" {
		t.Fatalf("hover = %+v", hover)
	}
}

func TestHoverFunctionDeclarationAndUse(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\nresult := square(5)\n"
	declaration := hoverAt(t, text, "file:///main.ahd", "square:", 1)
	if declaration.Contents.Value != "square: (value: Int) -> Int" {
		t.Fatalf("declaration hover = %+v", declaration)
	}
	use := hoverAt(t, text, "file:///main.ahd", "square(5)", 1)
	if use.Contents.Value != "square: (value: Int) -> Int" {
		t.Fatalf("use hover = %+v", use)
	}
}

func TestHoverParameter(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\n"
	hover := hoverAt(t, text, "file:///main.ahd", "value: Int", 1)
	if hover.Contents.Value != "value: Int" {
		t.Fatalf("hover = %+v", hover)
	}
}

func TestHoverImportedStandardModuleMember(t *testing.T) {
	text := "bring Math\nwrite(Math.PI)\n"
	hover := hoverAt(t, text, "file:///main.ahd", "Math.PI", len("Math.")+1)
	if hover.Contents.Value != "Constant PI: Real" {
		t.Fatalf("hover = %+v", hover)
	}
}

func TestHoverPositionWithNoSymbolReturnsNull(t *testing.T) {
	text := "x: Int := 1 + 2\n"
	if !hoverIsNull(t, text, "file:///main.ahd", "1 + 2", 0) {
		t.Fatal("expected a null hover result over a bare literal")
	}
}

func TestHoverUnicodeSourcePosition(t *testing.T) {
	text := "isim: String := \"Ayşe 🙂\"\nwrite(isim)\n"
	declaration := hoverAt(t, text, "file:///main.ahd", "isim", 1)
	if declaration.Contents.Value != "isim: String" {
		t.Fatalf("declaration hover = %+v", declaration)
	}
	use := hoverAt(t, text, "file:///main.ahd", "write(isim", len("write(")+1)
	if use.Contents.Value != "isim: String" {
		t.Fatalf("use hover = %+v", use)
	}
}

// TestHoverAfterDidChangeUsesNewSourceNotSavedDiskContent proves hover
// reflects the in-memory buffer, not whatever the file on disk still says.
func TestHoverAfterDidChangeUsesNewSourceNotSavedDiskContent(t *testing.T) {
	uri := "file:///main.ahd"
	firstText := "first: Int := 1\n"
	secondText := "renamed: Real := 2.5\n"

	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: firstText}})
	client.notify("textDocument/didChange", didChangeParams{
		TextDocument:   textDocumentIdentifier{URI: uri},
		ContentChanges: []contentChangeEvent{{Text: secondText}},
	})
	index := newLineIndex(secondText)
	position := index.OffsetToPosition(strings.Index(secondText, "renamed") + 1)
	client.request(2, "textDocument/hover", textDocumentPositionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("hover error: %+v", response.Error)
	}
	var hover lspHover
	if err := json.Unmarshal(response.Result, &hover); err != nil {
		t.Fatal(err)
	}
	if hover.Contents.Value != "renamed: Real" {
		t.Fatalf("hover after didChange = %+v, want the changed buffer's content", hover)
	}
}
