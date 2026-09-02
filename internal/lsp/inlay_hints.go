package lsp

import (
	"encoding/json"

	"ahdcode/internal/analysis"
)

type inlayHintParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

type inlayHintItem struct {
	Position     Position `json:"position"`
	Label        string   `json:"label"`
	Kind         int      `json:"kind,omitempty"`
	PaddingLeft  bool     `json:"paddingLeft,omitempty"`
	PaddingRight bool     `json:"paddingRight,omitempty"`
}

const (
	inlayHintKindType      = 1
	inlayHintKindParameter = 2
)

func (server *Server) handleInlayHint(m message) {
	var params inlayHintParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed inlayHint params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, []inlayHintItem{})
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.replyResult(m.ID, []inlayHintItem{})
		return
	}
	index := newLineIndex(text)
	hints := server.store.InlayHints(path)
	var converted []inlayHintItem
	for _, hint := range hints {
		position := index.OffsetToPosition(hint.Offset)
		item := inlayHintItem{
			Position: position,
			Label:    hint.Label,
		}
		switch hint.Kind {
		case analysis.InlayHintType:
			item.Kind = inlayHintKindType
			item.PaddingLeft = hint.Padding
		case analysis.InlayHintParameter:
			item.Kind = inlayHintKindParameter
			item.PaddingRight = hint.Padding
		}
		converted = append(converted, item)
	}
	server.replyResult(m.ID, converted)
}
