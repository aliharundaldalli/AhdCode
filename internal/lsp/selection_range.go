package lsp

import (
	"encoding/json"

	"ahdcode/internal/analysis"
)

type selectionRangeParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Positions    []Position             `json:"positions"`
}

type selectionRangeItem struct {
	Range  Range               `json:"range"`
	Parent *selectionRangeItem `json:"parent,omitempty"`
}

func (server *Server) handleSelectionRange(m message) {
	var params selectionRangeParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed selectionRange params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, []selectionRangeItem{})
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.replyResult(m.ID, []selectionRangeItem{})
		return
	}
	index := newLineIndex(text)
	converted := make([]selectionRangeItem, len(params.Positions))
	for i, position := range params.Positions {
		offset := index.PositionToOffset(position)
		head := server.store.SelectionRanges(path, offset)
		converted[i] = convertSelectionRange(index, head)
	}
	server.replyResult(m.ID, converted)
}

func convertSelectionRange(index *lineIndex, head *analysis.SelectionRange) selectionRangeItem {
	if head == nil {
		return selectionRangeItem{}
	}
	item := selectionRangeItem{
		Range: Range{
			Start: index.OffsetToPosition(head.StartOffset),
			End:   index.OffsetToPosition(head.EndOffset),
		},
	}
	if head.Parent != nil {
		parent := convertSelectionRange(index, head.Parent)
		item.Parent = &parent
	}
	return item
}
