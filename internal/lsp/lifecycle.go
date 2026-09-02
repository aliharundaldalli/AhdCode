package lsp

import (
	"encoding/json"

	"ahdcode/internal/analysis"
)

// textDocumentSyncFull is the LSP TextDocumentSyncKind value for full-
// document synchronization.
const textDocumentSyncFull = 1

// initializeParams captures workspace roots from the client.
type initializeParams struct {
	RootURI          *string `json:"rootUri"`
	WorkspaceFolders []struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	} `json:"workspaceFolders"`
}

type serverCapabilities struct {
	TextDocumentSync           int                   `json:"textDocumentSync"`
	HoverProvider              bool                  `json:"hoverProvider"`
	DefinitionProvider         bool                  `json:"definitionProvider"`
	DocumentSymbolProvider     bool                  `json:"documentSymbolProvider"`
	SignatureHelpProvider      signatureHelpOptions  `json:"signatureHelpProvider"`
	ReferencesProvider         bool                  `json:"referencesProvider"`
	CompletionProvider         completionOptions     `json:"completionProvider"`
	RenameProvider             bool                  `json:"renameProvider"`
	SemanticTokensProvider     semanticTokensOptions `json:"semanticTokensProvider"`
	InlayHintProvider          bool                  `json:"inlayHintProvider"`
	CodeActionProvider         codeActionOptions     `json:"codeActionProvider"`
	DocumentFormattingProvider bool                  `json:"documentFormattingProvider"`
	WorkspaceSymbolProvider    bool                  `json:"workspaceSymbolProvider"`
	FoldingRangeProvider       bool                  `json:"foldingRangeProvider"`
	SelectionRangeProvider     bool                  `json:"selectionRangeProvider"`
}

type codeActionOptions struct {
	CodeActionKinds []string `json:"codeActionKinds"`
}

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
}

type signatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type completionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

func (server *Server) handleInitialize(m message) {
	var params initializeParams
	_ = json.Unmarshal(m.Params, &params)
	var roots []string
	if params.RootURI != nil && *params.RootURI != "" {
		if path, err := URIToPath(*params.RootURI); err == nil {
			roots = append(roots, path)
		}
	}
	for _, folder := range params.WorkspaceFolders {
		if path, err := URIToPath(folder.URI); err == nil {
			roots = append(roots, path)
		}
	}
	server.store.SetWorkspaceRoots(roots)

	result := initializeResult{
		Capabilities: serverCapabilities{
			TextDocumentSync:           textDocumentSyncFull,
			HoverProvider:              true,
			DefinitionProvider:         true,
			DocumentSymbolProvider:     true,
			SignatureHelpProvider:      signatureHelpOptions{TriggerCharacters: []string{"(", ","}},
			ReferencesProvider:         true,
			CompletionProvider:         completionOptions{TriggerCharacters: []string{".", " "}},
			RenameProvider:             true,
			SemanticTokensProvider:     semanticTokensCapabilities(),
			InlayHintProvider:          true,
			CodeActionProvider:         codeActionOptions{CodeActionKinds: []string{"quickfix"}},
			DocumentFormattingProvider: true,
			WorkspaceSymbolProvider:    true,
			FoldingRangeProvider:       true,
			SelectionRangeProvider:     true,
		},
	}
	server.replyResult(m.ID, result)
}

// buildImportAdditionalEdit creates an additionalTextEdit for auto-import.
func buildImportAdditionalEdit(path, text string, importSpec *analysis.ImportEdit) (textEdit, bool) {
	if importSpec == nil {
		return textEdit{}, false
	}
	edit, ok := analysis.BuildImportEdit(path, text, *importSpec)
	if !ok {
		return textEdit{}, false
	}
	index := newLineIndex(text)
	return textEdit{
		Range: Range{
			Start: index.OffsetToPosition(edit.Span.Start.Offset),
			End:   index.OffsetToPosition(edit.Span.End.Offset),
		},
		NewText: edit.NewText,
	}, true
}
