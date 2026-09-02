package lsp

import "encoding/json"

type referenceParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      referenceContext       `json:"context"`
}

type referenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

func (server *Server) handleReferences(m message) {
	var params referenceParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed references params: "+err.Error())
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
	locations := server.store.References(path, offset, params.Context.IncludeDeclaration)
	if locations == nil {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	indexes := make(map[string]*lineIndex, len(locations))
	converted := make([]lspLocation, len(locations))
	for i, location := range locations {
		targetIndex, ok := indexes[location.Path]
		if !ok {
			targetText, _ := server.store.TextFor(path, location.Path)
			targetIndex = newLineIndex(targetText)
			indexes[location.Path] = targetIndex
		}
		converted[i] = lspLocation{
			URI: server.uriFor(location.Path),
			Range: Range{
				Start: targetIndex.OffsetToPosition(location.Span.Start.Offset),
				End:   targetIndex.OffsetToPosition(location.Span.End.Offset),
			},
		}
	}
	server.replyResult(m.ID, converted)
}
