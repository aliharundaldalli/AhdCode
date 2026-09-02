package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

func referencesAt(t *testing.T, text, uri, needle string, offsetWithinNeedle int, includeDeclaration bool) []lspLocation {
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
	client.request(2, "textDocument/references", referenceParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
		Context:      referenceContext{IncludeDeclaration: includeDeclaration},
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("references returned an error: %+v", response.Error)
	}
	var locations []lspLocation
	if err := json.Unmarshal(response.Result, &locations); err != nil {
		t.Fatalf("references result = %s: %v", response.Result, err)
	}
	return locations
}

func TestReferencesExcludesDeclarationByDefault(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\nwrite(score + 1.0)\n"
	locations := referencesAt(t, text, "file:///main.ahd", "score:", 1, false)
	if len(locations) != 2 {
		t.Fatalf("expected 2 references (declaration excluded), got %d: %#v", len(locations), locations)
	}
}

func TestReferencesIncludesDeclarationWhenRequested(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\n"
	locations := referencesAt(t, text, "file:///main.ahd", "score:", 1, true)
	if len(locations) != 2 {
		t.Fatalf("expected 2 references (declaration included), got %d: %#v", len(locations), locations)
	}
}

func TestReferencesOnBuiltinReturnsNull(t *testing.T) {
	text := "write(\"hi\")\n"
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: "file:///main.ahd", Text: text}})
	index := newLineIndex(text)
	client.request(2, "textDocument/references", referenceParams{
		TextDocument: textDocumentIdentifier{URI: "file:///main.ahd"},
		Position:     index.OffsetToPosition(1),
		Context:      referenceContext{IncludeDeclaration: true},
	})
	client.notify("exit", nil)
	messages := runServer(t, client)
	response := findResponse(t, messages, 2)
	if string(response.Result) != "null" {
		t.Fatalf("expected a null references result for a builtin, got %s", response.Result)
	}
}
