package analysis

import (
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
)

// SignatureHelp is one signature-help result: the callable's full rendered
// signature, its parameters rendered individually in the same style (so a
// caller can highlight just one), and which parameter the cursor is
// currently within.
type SignatureHelp struct {
	Label           string
	Parameters      []string
	ActiveParameter int
}

// SignatureHelp answers a signature-help request at the given byte offset:
// the signature of the nearest enclosing call the cursor is inside, using
// the exact Callable the compiler itself already selected for that call
// (semantic.Result.SelectedCallables) -- the same resolution a real call
// would use, including constructor calls and overload picks -- rather than
// re-deriving it from a name lookup. It reports ok=false when the cursor is
// not inside a call, or the call's callable never resolved (an unknown
// function, for instance).
func (store *Store) SignatureHelp(path string, offset int) (SignatureHelp, bool) {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()

	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return SignatureHelp{}, false
	}
	call := innermostCall(ancestorsAtOffset(entryModule.Parsed.Program, offset, cached.fileIDFor(canonical)))
	if call == nil {
		return SignatureHelp{}, false
	}
	callable, ok := entryModule.Semantic.SelectedCallables[call]
	if !ok || callable == nil || callable.Signature == nil {
		return SignatureHelp{}, false
	}
	parameters := semantic.FormatParameters(callable.Signature)
	active := activeParameterIndex(call, offset)
	if len(parameters) == 0 {
		// A zero-parameter callable has no valid parameter index at all;
		// 0 is the LSP-safe "nothing to highlight" value, never -1.
		active = 0
	} else if active >= len(parameters) {
		active = len(parameters) - 1
	}
	return SignatureHelp{
		Label:           semantic.FormatSignature(callable.Signature),
		Parameters:      parameters,
		ActiveParameter: active,
	}, true
}

// innermostCall returns the last (innermost) *ast.CallExpr in an ancestor
// path built by ancestorsAtOffset, or nil if the path never enters a call.
// The path's own outer-to-inner order already gives correct call-nesting
// preference: a cursor inside an inner call's arguments finds that inner
// call, while a cursor between two arguments of an outer call -- past a
// nested call's own closing paren, so descent never enters it -- finds the
// outer one, with no special-casing needed here.
func innermostCall(path []ast.Node) *ast.CallExpr {
	for index := len(path) - 1; index >= 0; index-- {
		if call, ok := path[index].(*ast.CallExpr); ok {
			return call
		}
	}
	return nil
}

// activeParameterIndex reports which argument slot offset falls in: the
// index of the argument whose own span contains it, or -- between two
// arguments, or past every argument written so far -- the index one past
// the last argument that already ends before it. This is a purely
// structural computation over already-parsed CallArgument spans, with no
// token-level comma scanning of its own.
func activeParameterIndex(call *ast.CallExpr, offset int) int {
	for index := range call.Arguments {
		argument := &call.Arguments[index]
		if containsOffset(argument.Span(), offset) {
			return index
		}
		if offset < argument.Span().Start.Offset {
			return index
		}
	}
	return len(call.Arguments)
}
