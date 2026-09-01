package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/analysis"
)

func latestDiagnostics(t *testing.T, messages []message, uri string) []lspDiagnostic {
	t.Helper()
	var latest *publishDiagnosticsParams
	for _, m := range findNotifications(messages, "textDocument/publishDiagnostics") {
		var params publishDiagnosticsParams
		if err := json.Unmarshal(m.Params, &params); err != nil {
			t.Fatal(err)
		}
		if params.URI == uri {
			copyParams := params
			latest = &copyParams
		}
	}
	if latest == nil {
		t.Fatalf("no publishDiagnostics notification for %s", uri)
	}
	return latest.Diagnostics
}

func TestDidOpenValidProgramHasNoDiagnostics(t *testing.T) {
	uri := "file:///main.ahd"
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: "write(\"hello\")\n"}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	diags := latestDiagnostics(t, messages, uri)
	if len(diags) != 0 {
		t.Fatalf("expected zero diagnostics, got %+v", diags)
	}
}

func TestDidOpenLexerErrorProducesDiagnostic(t *testing.T) {
	uri := "file:///main.ahd"
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: "write(\"unterminated)\n"}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	diags := latestDiagnostics(t, messages, uri)
	if len(diags) == 0 {
		t.Fatal("expected a lexer diagnostic")
	}
	if diags[0].Source != "ahdcode" {
		t.Fatalf("source = %q, want ahdcode", diags[0].Source)
	}
	if diags[0].Severity != diagnosticSeverityError {
		t.Fatalf("severity = %d, want %d", diags[0].Severity, diagnosticSeverityError)
	}
	if diags[0].Code == "" {
		t.Fatal("expected a stable diagnostic code")
	}
}

func TestDidChangeFixingTheErrorClearsDiagnostics(t *testing.T) {
	uri := "file:///main.ahd"
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: "x: Int := \"wrong\"\n"}})
	client.notify("textDocument/didChange", didChangeParams{
		TextDocument:   textDocumentIdentifier{URI: uri},
		ContentChanges: []contentChangeEvent{{Text: "x: Int := 5\n"}},
	})
	client.notify("exit", nil)
	messages := runServer(t, client)

	diags := latestDiagnostics(t, messages, uri)
	if len(diags) != 0 {
		t.Fatalf("expected diagnostics to clear after didChange, got %+v", diags)
	}
}

func TestDidCloseClearsOwnedDiagnostics(t *testing.T) {
	uri := "file:///main.ahd"
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: "x: Int := \"wrong\"\n"}})
	client.notify("textDocument/didClose", didCloseParams{TextDocument: textDocumentIdentifier{URI: uri}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	diags := latestDiagnostics(t, messages, uri)
	if len(diags) != 0 {
		t.Fatalf("expected didClose to clear diagnostics, got %+v", diags)
	}
}

func TestDidChangeAnalyzesUnsavedTextNeverWritesDisk(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(path, []byte("saved on disk\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	uri := PathToURI(path)
	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: uri, Text: "write(\"first\")\n"}})
	client.notify("textDocument/didChange", didChangeParams{
		TextDocument:   textDocumentIdentifier{URI: uri},
		ContentChanges: []contentChangeEvent{{Text: "write(\"second, unsaved\")\n"}},
	})
	client.notify("exit", nil)
	runServer(t, client)

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != "saved on disk\n" {
		t.Fatalf("the LSP server wrote to the real file: %q", onDisk)
	}
}

func TestImportedModuleDiagnosticsPublishedUnderItsOwnURI(t *testing.T) {
	directory := t.TempDir()
	entryPath := filepath.Join(directory, "main.ahd")
	helperPath := filepath.Join(directory, "Helper.ahd")
	if err := os.WriteFile(helperPath, []byte("x: Int := \"broken\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entryURI := PathToURI(entryPath)
	// Helper.ahd is only ever imported, never opened directly by the client,
	// so the server has no client-given URI for it and must fall back to
	// its canonical (symlink-resolved) path -- the same normalization
	// internal/module itself applies when resolving `bring Helper`.
	helperURI := PathToURI(analysis.CanonicalPath(helperPath))

	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: entryURI, Text: "bring Helper\nwrite(\"ok\")\n"}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	helperDiags := latestDiagnostics(t, messages, helperURI)
	if len(helperDiags) == 0 {
		t.Fatal("expected Helper.ahd's own diagnostic to be published under its own URI")
	}
}

// TestPublishedDiagnosticsUseTheExactClientURINotACanonicalizedOne is a
// regression test: internal/module canonicalizes paths with symlinks
// resolved (e.g. macOS's /var -> /private/var), but a client's own
// didOpen URI virtually never is. Publishing diagnostics under the
// canonicalized form instead of the client's own URI would create a
// phantom document VS Code never matches to the one it has open.
func TestPublishedDiagnosticsUseTheExactClientURINotACanonicalizedOne(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	clientURI := PathToURI(path)
	canonicalURI := PathToURI(analysis.CanonicalPath(path))

	client := &testClient{}
	client.request(1, "initialize", map[string]any{})
	client.notify("textDocument/didOpen", didOpenParams{TextDocument: textDocumentItem{URI: clientURI, Text: "x: Int := \"wrong\"\n"}})
	client.notify("exit", nil)
	messages := runServer(t, client)

	diags := latestDiagnostics(t, messages, clientURI)
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic published under the exact URI the client used")
	}
	if canonicalURI != clientURI {
		for _, m := range findNotifications(messages, "textDocument/publishDiagnostics") {
			var params publishDiagnosticsParams
			_ = json.Unmarshal(m.Params, &params)
			if params.URI == canonicalURI {
				t.Fatalf("a diagnostic was published under the canonicalized path %q instead of the client's own URI %q", canonicalURI, clientURI)
			}
		}
	}
}
