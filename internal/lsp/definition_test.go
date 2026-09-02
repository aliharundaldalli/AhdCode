package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

func definitionAt(t *testing.T, text, uri, needle string, offsetWithinNeedle int) (lspLocation, bool) {
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
	client.request(2, "textDocument/definition", textDocumentPositionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("definition returned an error: %+v", response.Error)
	}
	if string(response.Result) == "null" || len(response.Result) == 0 {
		return lspLocation{}, false
	}
	var location lspLocation
	if err := json.Unmarshal(response.Result, &location); err != nil {
		t.Fatalf("definition result = %s: %v", response.Result, err)
	}
	return location, true
}

func TestDefinitionOnUseJumpsToDeclaration(t *testing.T) {
	text := "score: Real := 85.0\nwrite(score)\n"
	location, ok := definitionAt(t, text, "file:///main.ahd", "write(score", len("write(")+1)
	if !ok {
		t.Fatal("expected a definition location")
	}
	if location.URI != "file:///main.ahd" {
		t.Fatalf("location.URI = %q", location.URI)
	}
	index := newLineIndex(text)
	want := index.OffsetToPosition(strings.Index(text, "score:"))
	if location.Range.Start != want {
		t.Fatalf("location.Range.Start = %+v, want %+v", location.Range.Start, want)
	}
}

func TestDefinitionOnBuiltinReturnsNull(t *testing.T) {
	text := "write(\"hi\")\n"
	if _, ok := definitionAt(t, text, "file:///main.ahd", "write", 1); ok {
		t.Fatal("expected a null definition result for a builtin")
	}
}
