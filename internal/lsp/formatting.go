package lsp

import "encoding/json"

type formattingParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Options      formattingOptions      `json:"options"`
}

type formattingOptions struct {
	TabSize      int  `json:"tabSize"`
	InsertSpaces bool `json:"insertSpaces"`
}

func (server *Server) handleFormatting(m message) {
	var params formattingParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed formatting params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, []textEdit{})
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.replyResult(m.ID, []textEdit{})
		return
	}
	edit, ok := server.store.FormatTextEdit(path)
	if !ok {
		server.replyResult(m.ID, []textEdit{})
		return
	}
	index := newLineIndex(text)
	server.replyResult(m.ID, []textEdit{{
		Range: Range{
			Start: index.OffsetToPosition(edit.Span.Start.Offset),
			End:   index.OffsetToPosition(edit.Span.End.Offset),
		},
		NewText: edit.NewText,
	}})
}
