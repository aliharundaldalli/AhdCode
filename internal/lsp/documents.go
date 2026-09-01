package lsp

import (
	"encoding/json"

	"ahdcode/internal/analysis"
)

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type textDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

// contentChangeEvent is one entry of textDocument/didChange's
// contentChanges array. Full document sync (the only kind v0.2.0
// advertises) never sets Range: every change event carries the document's
// entire new text.
type contentChangeEvent struct {
	Text string `json:"text"`
}

type didChangeParams struct {
	TextDocument   textDocumentIdentifier `json:"textDocument"`
	ContentChanges []contentChangeEvent   `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

func (server *Server) handleDidOpen(m message) {
	var params didOpenParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.logf("lsp: malformed didOpen params: %v", err)
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.logf("lsp: didOpen for an unsupported document URI %q: %v", params.TextDocument.URI, err)
		return
	}
	server.clientURI[analysis.CanonicalPath(path)] = params.TextDocument.URI
	result := server.store.Open(path, params.TextDocument.Text)
	server.publishDiagnostics(result)
}

func (server *Server) handleDidChange(m message) {
	var params didChangeParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.logf("lsp: malformed didChange params: %v", err)
		return
	}
	if len(params.ContentChanges) == 0 {
		return
	}
	// Full sync: the last event in the array is the document's complete
	// current text (a compliant client sends exactly one event per change
	// under TextDocumentSyncKind.Full, but taking the last one tolerates a
	// client that batches several).
	text := params.ContentChanges[len(params.ContentChanges)-1].Text
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.logf("lsp: didChange for an unsupported document URI %q: %v", params.TextDocument.URI, err)
		return
	}
	server.clientURI[analysis.CanonicalPath(path)] = params.TextDocument.URI
	result := server.store.Change(path, text)
	server.publishDiagnostics(result)
}

func (server *Server) handleDidClose(m message) {
	var params didCloseParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.logf("lsp: malformed didClose params: %v", err)
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.logf("lsp: didClose for an unsupported document URI %q: %v", params.TextDocument.URI, err)
		return
	}
	owned := server.store.Close(path)
	for _, ownedPath := range owned {
		server.publishOne(ownedPath, "", nil)
	}
	delete(server.published, analysis.CanonicalPath(path))
}
