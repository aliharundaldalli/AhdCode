package lsp

import "encoding/json"

type codeActionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      codeActionContext      `json:"context"`
}

type codeActionContext struct {
	Diagnostics []lspDiagnostic `json:"diagnostics"`
	Only        []string        `json:"only,omitempty"`
}

type codeActionItem struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Edit        *workspaceEdit `json:"edit,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
}

func (server *Server) handleCodeAction(m message) {
	var params codeActionParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed codeAction params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, []codeActionItem{})
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.replyResult(m.ID, []codeActionItem{})
		return
	}
	offset := newLineIndex(text).PositionToOffset(params.Range.Start)
	actions := server.store.CodeActions(path, offset)
	var converted []codeActionItem
	for _, action := range actions {
		changes := make(map[string][]textEdit)
		for _, edit := range action.Edits {
			targetText, _ := server.store.TextFor(path, edit.Path)
			targetIndex := newLineIndex(targetText)
			uri := server.uriFor(edit.Path)
			changes[uri] = append(changes[uri], textEdit{
				Range: Range{
					Start: targetIndex.OffsetToPosition(edit.Span.Start.Offset),
					End:   targetIndex.OffsetToPosition(edit.Span.End.Offset),
				},
				NewText: edit.NewText,
			})
		}
		converted = append(converted, codeActionItem{
			Title: action.Title,
			Kind:  "quickfix",
			Edit:  &workspaceEdit{Changes: changes},
		})
	}
	server.replyResult(m.ID, converted)
}
