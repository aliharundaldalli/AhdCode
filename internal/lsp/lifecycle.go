package lsp

// textDocumentSyncFull is the LSP TextDocumentSyncKind value for full-
// document synchronization: every didChange notification carries the
// document's complete new text, never an incremental edit. v0.2.0
// deliberately never advertises or implements incremental sync (kind 2).
const textDocumentSyncFull = 1

// initializeResult is the response to `initialize`. Its capabilities object
// is deliberately minimal: it must advertise exactly the features this
// server implements -- full document sync, diagnostics, hover, go-to-
// definition, document symbols, signature help, find references, and
// completion -- and nothing else, so a client never believes it can request
// rename, semantic tokens, or any other feature this release does not have.
type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
}

type serverCapabilities struct {
	TextDocumentSync       int                  `json:"textDocumentSync"`
	HoverProvider          bool                 `json:"hoverProvider"`
	DefinitionProvider     bool                 `json:"definitionProvider"`
	DocumentSymbolProvider bool                 `json:"documentSymbolProvider"`
	SignatureHelpProvider  signatureHelpOptions `json:"signatureHelpProvider"`
	ReferencesProvider     bool                 `json:"referencesProvider"`
	CompletionProvider     completionOptions    `json:"completionProvider"`
}

type signatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type completionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

func (server *Server) handleInitialize(m message) {
	result := initializeResult{
		Capabilities: serverCapabilities{
			TextDocumentSync:       textDocumentSyncFull,
			HoverProvider:          true,
			DefinitionProvider:     true,
			DocumentSymbolProvider: true,
			SignatureHelpProvider:  signatureHelpOptions{TriggerCharacters: []string{"(", ","}},
			ReferencesProvider:     true,
			CompletionProvider:     completionOptions{TriggerCharacters: []string{"."}},
		},
	}
	server.replyResult(m.ID, result)
}
