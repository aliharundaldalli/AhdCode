package lsp

// textDocumentSyncFull is the LSP TextDocumentSyncKind value for full-
// document synchronization: every didChange notification carries the
// document's complete new text, never an incremental edit. v0.2.0
// deliberately never advertises or implements incremental sync (kind 2).
const textDocumentSyncFull = 1

// initializeResult is the v0.2.0 response to `initialize`. Its
// capabilities object is deliberately minimal: it must advertise exactly
// the features this server implements (full document sync, diagnostics,
// hover) and nothing else, so a client never believes it can request
// completion, go-to-definition, or any other feature this release does not
// have.
type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
}

type serverCapabilities struct {
	TextDocumentSync int  `json:"textDocumentSync"`
	HoverProvider    bool `json:"hoverProvider"`
}

func (server *Server) handleInitialize(m message) {
	result := initializeResult{
		Capabilities: serverCapabilities{
			TextDocumentSync: textDocumentSyncFull,
			HoverProvider:    true,
		},
	}
	server.replyResult(m.ID, result)
}
