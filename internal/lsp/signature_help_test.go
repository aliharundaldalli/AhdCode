package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

func signatureHelpAt(t *testing.T, text, uri, needle string, offsetWithinNeedle int) (lspSignatureHelp, bool) {
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
	client.request(2, "textDocument/signatureHelp", textDocumentPositionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     position,
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	response := findResponse(t, messages, 2)
	if response.Error != nil {
		t.Fatalf("signatureHelp returned an error: %+v", response.Error)
	}
	if string(response.Result) == "null" || len(response.Result) == 0 {
		return lspSignatureHelp{}, false
	}
	var help lspSignatureHelp
	if err := json.Unmarshal(response.Result, &help); err != nil {
		t.Fatalf("signatureHelp result = %s: %v", response.Result, err)
	}
	return help, true
}

func TestSignatureHelpInsideCall(t *testing.T) {
	text := "square: Function := (\n    value: Int\n) -> Int {\n    return value * value\n}\nresult := square(5)\n"
	help, ok := signatureHelpAt(t, text, "file:///main.ahd", "square(5)", len("square("))
	if !ok {
		t.Fatal("expected signature help")
	}
	if len(help.Signatures) != 1 || help.Signatures[0].Label != "(value: Int) -> Int" {
		t.Fatalf("help = %+v", help)
	}
	if len(help.Signatures[0].Parameters) != 1 || help.Signatures[0].Parameters[0].Label != "value: Int" {
		t.Fatalf("parameters = %+v", help.Signatures[0].Parameters)
	}
	if help.ActiveParameter != 0 {
		t.Fatalf("activeParameter = %d", help.ActiveParameter)
	}
}

func TestSignatureHelpOutsideCallReturnsNull(t *testing.T) {
	text := "x: Int := 5\n"
	if _, ok := signatureHelpAt(t, text, "file:///main.ahd", "5", 0); ok {
		t.Fatal("expected a null signatureHelp result outside any call")
	}
}
