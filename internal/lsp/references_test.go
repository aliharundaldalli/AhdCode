package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/analysis"
)

func TestWorkspaceReferencesAndRenameUseClosedFileUTF16Positions(t *testing.T) {
	root := t.TempDir()
	entry, consumer := filepath.Join(root, "Answer.ahd"), filepath.Join(root, "Consumer.ahd")
	entryText := "answer: Int := 42\r\n"
	consumerText := "from Answer bring answer\r\nwrite(\"😀{answer}\")\r\n"
	for path, text := range map[string]string{entry: entryText, consumer: consumerText} {
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entryURI, consumerURI := PathToURI(entry), PathToURI(analysis.CanonicalPath(consumer))
	document := textDocumentIdentifier{URI: entryURI}
	position := Position{Character: 1}
	want := Range{Start: Position{Line: 1, Character: 10}, End: Position{Line: 1, Character: 16}}
	client := &testClient{}
	client.request(1, "initialize", map[string]any{"rootUri": PathToURI(root)})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: entryURI, Text: entryText}})
	client.request(2, "textDocument/references", referenceParams{TextDocument: document, Position: position})
	client.request(3, "textDocument/rename", renameParams{TextDocument: document, Position: position, NewName: "result"})
	client.notify("exit", nil)
	messages := runServer(t, client)
	var locations []lspLocation
	if err := json.Unmarshal(findResponse(t, messages, 2).Result, &locations); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, location := range locations {
		found = found || (location.URI == consumerURI && location.Range == want)
	}
	if !found {
		t.Fatalf("missing UTF-16 reference %v in %v", want, locations)
	}
	var edits workspaceEdit
	if err := json.Unmarshal(findResponse(t, messages, 3).Result, &edits); err != nil {
		t.Fatal(err)
	}
	found = false
	for _, edit := range edits.Changes[consumerURI] {
		found = found || (edit.Range == want && edit.NewText == "result")
	}
	if !found {
		t.Fatalf("missing UTF-16 edit %v in %v", want, edits)
	}
}

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
