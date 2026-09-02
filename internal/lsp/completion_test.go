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
