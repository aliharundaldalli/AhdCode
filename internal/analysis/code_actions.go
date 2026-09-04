package analysis

import (
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
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
			if action, ok := quickFixForDiagnostic(ownerPath, text, item); ok {
				actions = append(actions, action)
			}
		}
	}
	return actions
}

func quickFixForDiagnostic(path, text string, item diagnostics.Diagnostic) (CodeAction, bool) {
	switch item.Code {
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
