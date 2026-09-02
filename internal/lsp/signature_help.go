package lsp

import "encoding/json"

type lspSignatureHelp struct {
	Signatures      []lspSignatureInformation `json:"signatures"`
	ActiveSignature int                       `json:"activeSignature"`
	ActiveParameter int                       `json:"activeParameter"`
}

type lspSignatureInformation struct {
	Label      string                    `json:"label"`
	Parameters []lspParameterInformation `json:"parameters"`
}

type lspParameterInformation struct {
	Label string `json:"label"`
}

func (server *Server) handleSignatureHelp(m message) {
	var params textDocumentPositionParams
	if err := json.Unmarshal(m.Params, &params); err != nil {
		server.sendErrorResponse(m.ID, errCodeInvalidParams, "malformed signatureHelp params: "+err.Error())
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
	help, ok := server.store.SignatureHelp(path, offset)
	if !ok {
		server.reply(m.ID, json.RawMessage("null"))
		return
	}
	parameters := make([]lspParameterInformation, len(help.Parameters))
	for index, parameter := range help.Parameters {
		parameters[index] = lspParameterInformation{Label: parameter}
	}
	server.replyResult(m.ID, lspSignatureHelp{
		Signatures:      []lspSignatureInformation{{Label: help.Label, Parameters: parameters}},
		ActiveSignature: 0,
		ActiveParameter: help.ActiveParameter,
	})
}
