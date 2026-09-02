package analysis

import (
	"strings"

	"ahdcode/internal/formatter"
	"ahdcode/internal/source"
)

// FormatDocument formats path's open-buffer text using the canonical AhdCode
// formatter. It never writes to disk.
func (store *Store) FormatDocument(path string) (string, bool) {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	text, ok := store.documents[canonical]
	store.mutex.Unlock()
	if !ok {
		return "", false
	}
	file := source.File{Path: canonical, Text: text}
	result := formatter.Format(file)
	if result.HasErrors() || result.Text == "" {
		return "", false
	}
	return result.Text, true
}

// FormatTextEdit returns one whole-document TextEdit replacing path's content
// with the formatted text.
func (store *Store) FormatTextEdit(path string) (TextEdit, bool) {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	text, ok := store.documents[canonical]
	store.mutex.Unlock()
	if !ok {
		return TextEdit{}, false
	}
	formatted, ok := store.FormatDocument(path)
	if !ok {
		return TextEdit{}, false
	}
	if formatted == text {
		return TextEdit{}, false
	}
	return TextEdit{
		Path:    canonical,
		Span:    source.Span{Start: source.Position{Offset: 0}, End: source.Position{Offset: len(text)}},
		NewText: formatted,
	}, true
}

// BuildImportEdit constructs a TextEdit that adds or extends an import for
// one symbol from a user module.
func BuildImportEdit(path, text string, importSpec ImportEdit) (TextEdit, bool) {
	return buildImportEdit(path, text, importSpec)
}

func buildImportEdit(path, text string, importSpec ImportEdit) (TextEdit, bool) {
	if importSpec.ModuleName == "" || importSpec.SymbolName == "" {
		return TextEdit{}, false
	}
	lines := strings.Split(text, "\n")
	insertLine := 0
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "bring ") || strings.HasPrefix(trimmed, "from ") {
			insertLine = index + 1
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			insertLine = index + 1
			continue
		}
		break
	}
	importLine := "from " + importSpec.ModuleName + " bring " + importSpec.SymbolName + "\n"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		prefix := "from " + importSpec.ModuleName + " bring "
		if strings.HasPrefix(trimmed, prefix) {
			if strings.Contains(trimmed, importSpec.SymbolName) {
				return TextEdit{}, false
			}
			// Extend existing from-import list.
			extended := strings.TrimRight(trimmed, "\n") + ", " + importSpec.SymbolName
			start := strings.Index(text, line)
			if start < 0 {
				break
			}
			return TextEdit{
				Path:    path,
				Span:    source.Span{Start: source.Position{Offset: start}, End: source.Position{Offset: start + len(line)}},
				NewText: extended,
			}, true
		}
	}
	// Insert new import before first non-import statement.
	offset := 0
	for index := 0; index < insertLine && index < len(lines); index++ {
		offset += len(lines[index]) + 1
	}
	return TextEdit{
		Path:    path,
		Span:    source.Span{Start: source.Position{Offset: offset}, End: source.Position{Offset: offset}},
		NewText: importLine,
	}, true
}
