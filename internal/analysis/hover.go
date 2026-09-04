package analysis

import (
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/types"
)

// Hover is one hover result: the rendered text and the exact source span it
// describes (so a caller can highlight the token the hover applies to).
type Hover struct {
	Text string
	Span source.Span
}

// Hover answers a hover request at the given byte offset within the most
// recently analyzed snapshot of path. It reports ok=false whenever there is
// no confidently resolved semantic symbol at that position -- an unanalyzed
// document, a position with no symbol, or an expression the analyzer never
// gave a name -- rather than guess.
func (store *Store) Hover(path string, offset int) (Hover, bool) {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return Hover{}, false
	}
	node := findNodeAtOffset(entryModule.Parsed.Program, offset, cached.fileIDFor(canonical))
	if node == nil {
		return Hover{}, false
	}
	symbol, ok := entryModule.Semantic.ResolvedSymbols[node]
	if !ok || symbol == nil {
		return Hover{}, false
	}
	return Hover{Text: renderHover(symbol), Span: node.Span()}, true
}

// renderHover turns an already-resolved compiler Symbol into hover text,
// reusing the compiler's own type/signature renderers (types.Display,
// semantic.FormatSignature) verbatim rather than inventing a second one.
func renderHover(symbol *semantic.Symbol) string {
	if symbol.Callable != nil && symbol.Callable.Signature != nil {
		return symbol.Name + ": " + semantic.FormatSignature(symbol.Callable.Signature)
	}
	switch symbol.Kind {
	case semantic.ClassSymbol:
		return "Class " + symbol.Name
	case semantic.NamespaceSymbol:
		return "module " + symbol.Name
	}
	prefix := ""
	if symbol.Constant {
		prefix = "Constant "
	}
	return prefix + symbol.Name + ": " + types.Display(symbol.Type)
}
