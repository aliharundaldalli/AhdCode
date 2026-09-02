package lsp

import "encoding/json"

type workspaceSymbolParams struct {
	Query string `json:"query"`
}

type workspaceSymbolItem struct {
	Name          string `json:"name"`
	Kind          int    `json:"kind"`
	ContainerName string `json:"containerName,omitempty"`
	Location      lspLocation `json:"location"`
}

func (server *Server) handleWorkspaceSymbol(m message) {
	var params workspaceSymbolParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed workspaceSymbol params: "+err.Error())
		return
	}
	entryPath := server.primaryEntryPath()
	if entryPath == "" {
		server.replyResult(m.ID, []workspaceSymbolItem{})
		return
	}
	symbols := server.store.WorkspaceSymbols(entryPath, params.Query)
	converted := make([]workspaceSymbolItem, 0, len(symbols))
	for _, symbol := range symbols {
		targetText, _ := server.store.TextFor(entryPath, symbol.Path)
		if targetText == "" {
			continue
		}
		startOffset := symbol.StartOffset
		endOffset := symbol.EndOffset
		if endOffset <= startOffset {
			startOffset = 0
			endOffset = 0
		}
		index := newLineIndex(targetText)
		converted = append(converted, workspaceSymbolItem{
			Name:          symbol.Name,
			Kind:          lspSymbolKind(symbol.Kind, false),
			ContainerName: symbol.ModuleName,
			Location: lspLocation{
				URI: server.uriFor(symbol.Path),
				Range: Range{
					Start: index.OffsetToPosition(startOffset),
					End:   index.OffsetToPosition(endOffset),
				},
			},
		})
	}
	server.replyResult(m.ID, converted)
}

func (server *Server) primaryEntryPath() string {
	return server.store.PrimaryOpenPath()
}
