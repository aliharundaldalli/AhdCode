package lsp

import "encoding/json"

type foldingRangeItem struct {
	StartLine      int    `json:"startLine"`
	StartCharacter int    `json:"startCharacter,omitempty"`
	EndLine        int    `json:"endLine"`
	EndCharacter   int    `json:"endCharacter,omitempty"`
	Kind           string `json:"kind,omitempty"`
}

func (server *Server) handleFoldingRange(m message) {
	var params textDocumentParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed foldingRange params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, []foldingRangeItem{})
		return
	}
	if _, ok := server.store.Text(path); !ok {
		server.replyResult(m.ID, []foldingRangeItem{})
		return
	}
	ranges := server.store.FoldingRanges(path)
	converted := make([]foldingRangeItem, len(ranges))
	for index, item := range ranges {
		converted[index] = foldingRangeItem{
			StartLine: item.StartLine,
			EndLine:   item.EndLine,
			Kind:      item.Kind,
		}
	}
	server.replyResult(m.ID, converted)
}
