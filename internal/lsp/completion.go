package lsp

import "encoding/json"

type lspCompletionItem struct {
	Label               string     `json:"label"`
	Detail              string     `json:"detail,omitempty"`
	AdditionalTextEdits []textEdit `json:"additionalTextEdits,omitempty"`
}

func (server *Server) handleCompletion(m message) {
	var params textDocumentPositionParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed completion params: "+err.Error())
		return
	}
	path, err := URIToPath(params.TextDocument.URI)
	if err != nil {
		server.replyResult(m.ID, []lspCompletionItem{})
		return
	}
	text, ok := server.store.Text(path)
	if !ok {
		server.replyResult(m.ID, []lspCompletionItem{})
		return
	}
	offset := newLineIndex(text).PositionToOffset(params.Position)
	items := server.store.Completion(path, offset)
	converted := make([]lspCompletionItem, len(items))
	for index, item := range items {
		entry := lspCompletionItem{Label: item.Label, Detail: item.Detail}
		if item.Import != nil {
			if importEdit, ok := buildImportAdditionalEdit(path, text, item.Import); ok {
				entry.AdditionalTextEdits = []textEdit{importEdit}
			}
		}
		converted[index] = entry
	}
	server.replyResult(m.ID, converted)
}
