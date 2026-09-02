package lsp

import (
	"encoding/json"

	"ahdcode/internal/analysis"
	"ahdcode/internal/semantic"
)

type documentSymbolParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

// lspDocumentSymbol is the LSP hierarchical DocumentSymbol shape.
// SelectionRange repeats Range: AhdCode's AST does not carry a separate
// sub-span for a declaration's own name token (FunctionDecl.Name and
// ClassDecl.Name are bare strings, not nodes with their own Span), so the
// whole declaration's range is the most precise span actually available.
type lspDocumentSymbol struct {
	Name           string              `json:"name"`
	Detail         string              `json:"detail,omitempty"`
	Kind           int                 `json:"kind"`
	Range          Range               `json:"range"`
	SelectionRange Range               `json:"selectionRange"`
	Children       []lspDocumentSymbol `json:"children,omitempty"`
}

func (server *Server) handleDocumentSymbol(m message) {
	var params documentSymbolParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed documentSymbol params: "+err.Error())
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
	index := newLineIndex(text)
	symbols := server.store.DocumentSymbols(path)
	server.replyResult(m.ID, convertDocumentSymbols(symbols, index, false))
}

// convertDocumentSymbols converts one level of the outline. isMember is true
// for a Class's own children: semantic.FunctionSymbol covers both a
// top-level function and a Class method identically (the compiler has no
// separate "method" SymbolKind), so only the outline's own structure --
// whether a symbol sits inside a Class -- can tell the two apart for the
// LSP Method vs. Function icon.
func convertDocumentSymbols(symbols []analysis.DocumentSymbol, index *lineIndex, isMember bool) []lspDocumentSymbol {
	converted := make([]lspDocumentSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		converted = append(converted, convertDocumentSymbol(symbol, index, isMember))
	}
	return converted
}

func convertDocumentSymbol(symbol analysis.DocumentSymbol, index *lineIndex, isMember bool) lspDocumentSymbol {
	span := Range{
		Start: index.OffsetToPosition(symbol.Span.Start.Offset),
		End:   index.OffsetToPosition(symbol.Span.End.Offset),
	}
	return lspDocumentSymbol{
		Name:           symbol.Name,
		Detail:         symbol.Detail,
		Kind:           lspSymbolKind(symbol.Kind, isMember),
		Range:          span,
		SelectionRange: span,
		Children:       convertDocumentSymbols(symbol.Children, index, true),
	}
}

// LSP SymbolKind values (the subset this server ever produces).
const (
	symbolKindNamespace = 3
	symbolKindClass     = 5
	symbolKindMethod    = 6
	symbolKindField     = 8
	symbolKindFunction  = 12
	symbolKindVariable  = 13
)

func lspSymbolKind(kind semantic.SymbolKind, isMember bool) int {
	switch kind {
	case semantic.ClassSymbol:
		return symbolKindClass
	case semantic.FunctionSymbol:
		if isMember {
			return symbolKindMethod
		}
		return symbolKindFunction
	case semantic.MemberSymbol:
		return symbolKindField
	case semantic.NamespaceSymbol:
		return symbolKindNamespace
	default:
		return symbolKindVariable
	}
}
