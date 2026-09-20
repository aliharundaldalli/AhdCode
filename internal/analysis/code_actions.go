package analysis

import (
	"fmt"
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

// CodeAction is one deterministic quick fix tied to a compiler diagnostic.
type CodeAction struct {
	Title string
	Edits []TextEdit
}

const (
	codeMissingLocal         = "SEM006"
	codeScopeModifier        = "SEM005"
	codeInvalidControlSyntax = "PAR009"
	codeMissingCapture       = "SEM043"
	codeHiddenGlobal         = "SEM007"
)

// CodeActions returns quick fixes available at offset in path.
func (store *Store) CodeActions(path string, offset int) []CodeAction {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	if cached == nil {
		return nil
	}
	var actions []CodeAction
	for ownerPath, items := range cached.result.Diagnostics {
		if ownerPath != canonical {
			continue
		}
		text := cached.result.Text[ownerPath]
		fileID := cached.fileIDFor(canonical)
		for _, item := range items {
			if item.Severity != diagnostics.SeverityError {
				continue
			}
			if !containsOffsetInFile(item.Span, offset, fileID) {
				continue
			}
			if action, ok := quickFixForDiagnostic(ownerPath, text, item, cached); ok {
				actions = append(actions, action)
			}
		}
	}
	return actions
}

func quickFixForDiagnostic(path, text string, item diagnostics.Diagnostic, cached *entry) (CodeAction, bool) {
	switch item.Code {
	case codeMissingCapture:
		return missingCaptureFix(path, text, item, cached, '#')
	case codeHiddenGlobal:
		if strings.Contains(item.Message, "requires an explicit Global") {
			return missingCaptureFix(path, text, item, cached, '@')
		}
	case codeMissingLocal:
		return missingLocalFix(path, text, item)
	case codeInvalidControlSyntax:
		if strings.Contains(item.Message, "for iteration bindings are implicitly Local") {
			return removeForLocalFix(path, text, item)
		}
	case semantic.CodeExportNotFound:
		return unresolvedImportFix(path, text, item)
	}
	return CodeAction{}, false
}

func missingCaptureFix(path, text string, item diagnostics.Diagnostic, cached *entry, sigil byte) (CodeAction, bool) {
	name := quotedDiagnosticName(item.Message)
	if name == "" || cached == nil || cached.entryModule() == nil {
		return CodeAction{}, false
	}
	ancestors := ancestorsAtOffset(cached.entryModule().Parsed.Program, item.Span.Start.Offset, cached.fileIDFor(path))
	var function *ast.FunctionDecl
	for _, node := range ancestors {
		if candidate, ok := node.(*ast.FunctionDecl); ok {
			function = candidate
		} else if _, ok := node.(*ast.LambdaExpr); ok {
			function = nil
		}
	}
	if function == nil || function.Body == nil {
		return CodeAction{}, false
	}
	bodyStart := function.Body.Span().Start.Offset
	if bodyStart < 0 || bodyStart > len(text) {
		return CodeAction{}, false
	}
	itemText := string([]byte{sigil}) + name
	header := text[function.Span().Start.Offset:bodyStart]
	uses := strings.LastIndex(header, "uses")
	if uses >= 0 {
		openRelative := strings.Index(header[uses+len("uses"):], "[")
		if openRelative >= 0 {
			open := function.Span().Start.Offset + uses + len("uses") + openRelative
			closeRelative := strings.Index(header[uses+len("uses")+openRelative+1:], "]")
			if closeRelative >= 0 {
				close := open + 1 + closeRelative
				inside := text[open+1 : close]
				if strings.Contains(inside, itemText) || strings.Contains(inside, string([]byte{sigil})+name) {
					return CodeAction{}, false
				}
				insert := itemText
				if strings.TrimSpace(inside) != "" {
					insert = ", " + itemText
				}
				return CodeAction{Title: fmt.Sprintf("Add '%s' to uses list", itemText), Edits: []TextEdit{{
					Path: path, Span: source.Span{FileID: item.Span.FileID, Start: source.Position{Offset: close}, End: source.Position{Offset: close}}, NewText: insert,
				}}}, true
			}
		}
	}
	return CodeAction{Title: fmt.Sprintf("Add '%s' to uses list", itemText), Edits: []TextEdit{{
		Path: path, Span: source.Span{FileID: item.Span.FileID, Start: source.Position{Offset: bodyStart}, End: source.Position{Offset: bodyStart}}, NewText: "uses [" + itemText + "]\n",
	}}}, true
}

func quotedDiagnosticName(message string) string {
	start := strings.Index(message, "\"")
	if start < 0 {
		return ""
	}
	rest := message[start+1:]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func missingLocalFix(path, text string, item diagnostics.Diagnostic) (CodeAction, bool) {
	span := item.Span
	if span.Start.Offset >= len(text) {
		return CodeAction{}, false
	}
	declarationText := text[span.Start.Offset:]
	if span.End.Offset <= len(text) {
		declarationText = text[span.Start.Offset:span.End.Offset]
	}
	colonIndex := strings.Index(declarationText, ":")
	if colonIndex < 0 {
		return CodeAction{}, false
	}
	insertAt := span.Start.Offset + colonIndex + 1
	edit := TextEdit{
		Path:    path,
		Span:    source.Span{Start: source.Position{Offset: insertAt}, End: source.Position{Offset: insertAt}},
		NewText: " Local",
	}
	return CodeAction{Title: "Add Local modifier", Edits: []TextEdit{edit}}, true
}

func removeForLocalFix(path, text string, item diagnostics.Diagnostic) (CodeAction, bool) {
	// Remove erroneous Local/Global/Constant modifiers after for name:
	afterFor := text[item.Span.Start.Offset:]
	localIndex := strings.Index(afterFor, "Local")
	if localIndex < 0 {
		return CodeAction{}, false
	}
	start := item.Span.Start.Offset + localIndex
	end := start + len("Local")
	if start > 0 && text[start-1] == ' ' {
		start--
	}
	edit := TextEdit{
		Path:    path,
		Span:    source.Span{Start: source.Position{Offset: start}, End: source.Position{Offset: end}},
		NewText: "",
	}
	return CodeAction{Title: "Remove invalid Local from for binding", Edits: []TextEdit{edit}}, true
}

func unresolvedImportFix(path, text string, item diagnostics.Diagnostic) (CodeAction, bool) {
	// Extract quoted symbol name from message: module X has no symbol "Y"
	quoteStart := strings.Index(item.Message, "\"")
	if quoteStart < 0 {
		return CodeAction{}, false
	}
	rest := item.Message[quoteStart+1:]
	quoteEnd := strings.Index(rest, "\"")
	if quoteEnd < 0 {
		return CodeAction{}, false
	}
	symbolName := rest[:quoteEnd]
	moduleName := importModuleFromMessage(item.Message)
	if moduleName == "" || symbolName == "" {
		return CodeAction{}, false
	}
	importEdit, ok := buildImportEdit(path, text, ImportEdit{ModuleName: moduleName, SymbolName: symbolName})
	if !ok {
		importEdit, ok = removeInvalidFromImportLine(path, text, moduleName, symbolName)
	}
	if !ok {
		return CodeAction{}, false
	}
	return CodeAction{
		Title: "Import " + symbolName + " from " + moduleName,
		Edits: []TextEdit{importEdit},
	}, true
}

func importModuleFromMessage(message string) string {
	if index := strings.Index(message, "module "); index >= 0 {
		rest := message[index+len("module "):]
		if space := strings.IndexAny(rest, " \","); space > 0 {
			return rest[:space]
		}
	}
	return ""
}

func removeInvalidFromImportLine(path, text, moduleName, symbolName string) (TextEdit, bool) {
	want := "from " + moduleName + " bring " + symbolName
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != want {
			continue
		}
		start := strings.Index(text, line)
		if start < 0 {
			continue
		}
		end := start + len(line)
		if end < len(text) && text[end] == '\n' {
			end++
		}
		return TextEdit{
			Path:    path,
			Span:    source.Span{Start: source.Position{Offset: start}, End: source.Position{Offset: end}},
			NewText: "",
		}, true
	}
	return TextEdit{}, false
}
