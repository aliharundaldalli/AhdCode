package lsp

import "encoding/json"

type renameParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

type prepareRenameResult struct {
	Range       Range  `json:"range"`
	Placeholder string `json:"placeholder,omitempty"`
}

type workspaceEdit struct {
	Changes map[string][]textEdit `json:"changes,omitempty"`
}

type textEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

func (server *Server) handlePrepareRename(m message) {
	var params renameParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed prepareRename params: "+err.Error())
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
	span, ok := server.store.PrepareRename(path, offset)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	index := newLineIndex(text)
	server.replyResult(m.ID, prepareRenameResult{
		Range: Range{
			Start: index.OffsetToPosition(span.Start.Offset),
			End:   index.OffsetToPosition(span.End.Offset),
		},
	})
}

func (server *Server) handleRename(m message) {
	var params renameParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed rename params: "+err.Error())
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
	edits, ok := server.store.Rename(path, offset, params.NewName)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	changes := make(map[string][]textEdit)
	for _, edit := range edits {
		uri := server.uriFor(edit.Path)
		targetText, _ := server.store.TextFor(path, edit.Path)
		targetIndex := newLineIndex(targetText)
		changes[uri] = append(changes[uri], textEdit{
			Range: Range{
				Start: targetIndex.OffsetToPosition(edit.Span.Start.Offset),
				End:   targetIndex.OffsetToPosition(edit.Span.End.Offset),
			},
			NewText: edit.NewText,
		})
	}
	server.replyResult(m.ID, workspaceEdit{Changes: changes})
}
