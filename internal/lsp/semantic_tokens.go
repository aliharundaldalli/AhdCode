package lsp

import (
	"encoding/json"

	"ahdcode/internal/analysis"
)

type semanticTokensResult struct {
	Data []uint32 `json:"data"`
}

type semanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type semanticTokensOptions struct {
	Legend semanticTokensLegend `json:"legend"`
}

func semanticTokensCapabilities() semanticTokensOptions {
	return semanticTokensOptions{Legend: semanticTokensLegend{
		TokenTypes: []string{
			"namespace", "type", "function", "method", "parameter",
			"variable", "property", "keyword", "string", "number", "comment",
		},
		TokenModifiers: []string{"declaration", "readonly"},
	}}
}

func (server *Server) handleSemanticTokensFull(m message) {
	var params textDocumentParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed semanticTokens params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, semanticTokensResult{Data: []uint32{}})
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.replyResult(m.ID, semanticTokensResult{Data: []uint32{}})
		return
	}
	tokens := server.store.SemanticTokens(path)
	server.replyResult(m.ID, semanticTokensResult{Data: analysis.EncodeSemanticTokens(text, tokens)})
}
