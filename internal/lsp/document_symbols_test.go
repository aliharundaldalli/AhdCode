package lsp

import (
	"encoding/json"
	"testing"
)

func documentSymbolsFor(t *testing.T, text, uri string) []lspDocumentSymbol {
	t.Helper()
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: text}})
	client.request(2, "textDocument/documentSymbol", documentSymbolParams{TextDocument: textDocumentIdentifier{URI: uri}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("documentSymbol returned an error: %+v", response.Error)
	}
	var symbols []lspDocumentSymbol
	if err := json.Unmarshal(response.Result, &symbols); err != nil {
		t.Fatalf("documentSymbol result = %s: %v", response.Result, err)
	}
	return symbols
}

func TestDocumentSymbolListsTopLevelDeclarations(t *testing.T) {
	text := "score: Real := 85.0\n" +
		"square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\n"
	symbols := documentSymbolsFor(t, text, "file:///main.ahd")
	if len(symbols) != 2 {
		t.Fatalf("expected 2 top-level symbols, got %d: %#v", len(symbols), symbols)
	}
	if symbols[0].Name != "score" || symbols[0].Kind != symbolKindVariable {
		t.Fatalf("symbols[0] = %+v", symbols[0])
	}
	if symbols[1].Name != "square" || symbols[1].Kind != symbolKindFunction {
		t.Fatalf("symbols[1] = %+v", symbols[1])
	}
}

func TestDocumentSymbolClassMembersUseMethodAndFieldKinds(t *testing.T) {
	text := "Student: Class<> := {\n" +
		"    structure: Attributes := (\n        name: String\n    )\n\n" +
		"    describe: Function := (\n    ) -> String {\n        return attribute.name\n    }\n" +
		"}\n"
	symbols := documentSymbolsFor(t, text, "file:///main.ahd")
	if len(symbols) != 1 || symbols[0].Name != "Student" || symbols[0].Kind != symbolKindClass {
		t.Fatalf("expected one Class symbol, got %#v", symbols)
	}
	children := symbols[0].Children
	if len(children) != 2 {
		t.Fatalf("expected 2 Class members, got %d: %#v", len(children), children)
	}
	if children[0].Name != "name" || children[0].Kind != symbolKindField {
		t.Fatalf("children[0] = %+v", children[0])
	}
	if children[1].Name != "describe" || children[1].Kind != symbolKindMethod {
		t.Fatalf("children[1] = %+v", children[1])
	}
}
