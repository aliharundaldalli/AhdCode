package lsp

import "encoding/json"

// lspLocation is the LSP Location shape: a URI plus a range within it. It is
// distinct from analysis.Location, which names a file by canonical path
// (only meaningful within one compile) rather than a client-facing URI.
type lspLocation struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

func (server *Server) handleDefinition(m message) {
	var params textDocumentPositionParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed definition params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	offset := newLineIndex(text).PositionToOffset(params.Position)
	location, ok := server.store.Definition(path, offset)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	targetText, _ := server.store.TextFor(path, location.Path)
	targetIndex := newLineIndex(targetText)
	server.replyResult(m.ID, lspLocation{
		URI: server.uriFor(location.Path),
		Range: Range{
			Start: targetIndex.OffsetToPosition(location.Span.Start.Offset),
			End:   targetIndex.OffsetToPosition(location.Span.End.Offset),
		},
	})
}
