package semantic

import (
	"fmt"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// Explicit lexical capture.
//
// A lambda body is one expression, and it may read a binding from an enclosing
// callable only by listing that binding in its capture list. Capture is never
// inferred: an unlisted enclosing local is a diagnostic, so what a lambda
// depends on is visible where the lambda is written.
//
// A capture is resolved statically here and becomes a read-only binding inside
// the lambda with the enclosing binding's exact type, so nothing about closures
// weakens the type system.

// analyzeLambdaCaptures resolves one lambda's capture list and installs each
// captured name in the lambda's own scope. It runs before the parameters are
// installed so a capture that collides with a parameter is reported against the
// capture that caused it.
func (a *analyzer) analyzeLambdaCaptures(lambda *ast.LambdaExpr, current, lambdaScope *scope, flow, lambdaFlow flowState, callable *Callable) {
	parameterNames := make(map[string]bool, len(lambda.Parameters))
	for index := range lambda.Parameters {
		parameterNames[lambda.Parameters[index].Name] = true
	}
	seen := make(map[string]bool, len(lambda.Captures))
	for index := range lambda.Captures {
		capture := &lambda.Captures[index]
		if seen[capture.Name] {
			a.error(codeInvalidCapture, fmt.Sprintf("duplicate lambda capture %q", capture.Name),
				capture.Span(), "list each captured name once")
			continue
		}
		seen[capture.Name] = true
		if parameterNames[capture.Name] {
			a.error(codeInvalidCapture,
				fmt.Sprintf("captured name %q collides with a lambda parameter", capture.Name),
				capture.Span(), "rename the parameter or the captured binding; a lambda cannot bind one name twice")
			continue
		}
		outer, owner := current.lookup(capture.Name)
		if outer == nil {
			a.error(codeUnknownCapture, fmt.Sprintf("unknown capture %q", capture.Name),
				capture.Span(), "capture a binding that is visible where the lambda is written")
			continue
		}
		if !capturableSymbol(outer, owner, a.module) {
			a.error(codeInvalidCapture,
				fmt.Sprintf("%q is not a capturable lexical binding", capture.Name),
				capture.Span(), captureRejectionHint(outer, owner, a.module))
			continue
		}
		inner := &Symbol{
			Name: capture.Name, Kind: outer.Kind, Type: outer.Type,
			Span: capture.Span(), Declaration: capture,
			Confidential: outer.Confidential, Constant: outer.Constant,
			InitialNull: flow.state(outer), DeclaredNullable: outer.DeclaredNullable,
			OriginModuleID: outer.OriginModuleID,
			// A capture reads the enclosing binding's value. Rebinding it from
			// inside the lambda would suggest ownership of the outer variable,
			// which explicit capture deliberately does not grant.
			Captured: true,
		}
		lambdaScope.symbols[capture.Name] = inner
		lambdaFlow[inner] = inner.InitialNull
		a.result.Symbols = append(a.result.Symbols, inner)
		a.result.ResolvedSymbols[capture] = inner
		callable.Captures = append(callable.Captures, &Capture{Name: capture.Name, Outer: outer, Inner: inner})
	}
}

// capturableSymbol reports whether one resolved name may be captured. Only a
// binding belonging to an enclosing callable qualifies: a module-root name is
// reached by ordinary lookup and needs no closure storage, and a Class,
// namespace, or callable declaration is not a value binding at all.
func capturableSymbol(symbol *Symbol, owner, module *scope) bool {
	if symbol == nil || owner == module || symbol.Alias != nil {
		return false
	}
	if symbol.Builtin || symbol.ModuleRoot || symbol.SuperClassBinding {
		return false
	}
	return isLexicalCapture(symbol.Kind)
}

func captureRejectionHint(symbol *Symbol, owner, module *scope) string {
	if owner == module || symbol.ModuleRoot || symbol.Builtin {
		return fmt.Sprintf("%q is reachable by ordinary lookup, so remove it from the capture list", symbol.Name)
	}
	switch symbol.Kind {
	case ClassSymbol, NamespaceSymbol, FunctionSymbol:
		return fmt.Sprintf("%q is a declaration rather than a value binding; use it directly", symbol.Name)
	default:
		return "capture an enclosing Function parameter or Local binding"
	}
}

// captureBinding reports the capture entry that already covers one name inside
// a lambda, so reading it is accepted and reading an unlisted enclosing local
// is not.
func (callable *Callable) captureBinding(name string) *Capture {
	if callable == nil {
		return nil
	}
	for _, capture := range callable.Captures {
		if capture.Name == name {
			return capture
		}
	}
	return nil
}

// captureTypeText renders a captured binding's type for a diagnostic.
func captureTypeText(symbol *Symbol) string {
	if symbol == nil || symbol.Type == nil {
		return "value"
	}
	return types.Display(symbol.Type)
}
