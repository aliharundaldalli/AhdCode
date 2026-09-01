package lsp

import "encoding/json"

type textDocumentPositionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// lspHover is textDocument/hover's result. contents uses the plain
// "MarkupContent" shape (kind + value) rather than the older deprecated
// string/MarkedString forms.
type lspHover struct {
	Contents markupContent `json:"contents"`
	Range    Range         `json:"range"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

func (server *Server) handleHover(m message) {
	var params textDocumentPositionParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed hover params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		// An unsupported document scheme has no hover; this is not an error
		// worth failing the request over.
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	index := newLineIndex(text)
	offset := index.PositionToOffset(params.Position)
	found, ok := server.store.Hover(path, offset)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	server.replyResult(m.ID, lspHover{
		Contents: markupContent{Kind: "plaintext", Value: found.Text},
		Range: Range{
			Start: index.OffsetToPosition(found.Span.Start.Offset),
			End:   index.OffsetToPosition(found.Span.End.Offset),
		},
	})
}
