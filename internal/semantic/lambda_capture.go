package semantic

import (
	"fmt"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// Explicit lambda dependencies.
//
// A lambda body is one expression, and it may read a binding from outside its
// own parameters only by listing that binding in its dependency list.
// Dependency is never inferred: an unlisted enclosing local or module binding
// is a diagnostic, so what a lambda depends on is visible where the lambda is
// written.
//
// The list distinguishes two kinds of dependency, spelled either compactly or
// in full:
//
//   - Local capture (`#name` / `Local name`): reads an enclosing lexical
//     binding -- a Function parameter, a Local, a for-binding, or an
//     except-binding -- by value at the moment the lambda value is created.
//     The captured binding becomes a read-only binding inside the lambda with
//     the enclosing binding's exact type; nothing about it weakens the type
//     system, and it never becomes a hidden mutable environment.
//
//   - Global dependency (`@name` / `Global name`): declares that the lambda
//     intentionally reads a module/global binding, mirroring the explicit
//     Global declaration an ordinary Function already needs to touch module
//     state. It is not a capture: the dependency list entry installs an alias
//     to the real module binding, exactly like a Function's `x: Global Type`
//     declaration, so the lambda observes the live global rather than a
//     snapshot taken at lambda-creation time.
//
// `#name` and `Local name` are the same dependency under two spellings, and
// so are `@name` and `Global name`; duplicate detection treats them as one
// entry regardless of which spelling was used.

// analyzeLambdaCaptures resolves one lambda's dependency list and installs
// each entry in the lambda's own scope. It runs before the parameters are
// installed so a dependency that collides with a parameter is reported
// against the dependency that caused it.
func (a *analyzer) analyzeLambdaCaptures(lambda *ast.LambdaExpr, current, lambdaScope *scope, flow, lambdaFlow flowState, callable *Callable) {
	parameterNames := make(map[string]bool, len(lambda.Parameters))
	for index := range lambda.Parameters {
		parameterNames[lambda.Parameters[index].Name] = true
	}
	seen := make(map[string]bool, len(lambda.Captures))
	for index := range lambda.Captures {
		capture := &lambda.Captures[index]
		if seen[capture.Name] {
			a.error(codeInvalidCapture, fmt.Sprintf("%q is already listed as a lambda dependency", capture.Name),
				capture.Span(), "list each dependency once; #name/Local name and @name/Global name are alternate spellings of the same dependency")
			continue
		}
		seen[capture.Name] = true
		if parameterNames[capture.Name] {
			a.error(codeInvalidCapture,
				fmt.Sprintf("dependency %q collides with a lambda parameter", capture.Name),
				capture.Span(), "rename the parameter or the dependency; a lambda cannot bind one name twice")
			continue
		}
		switch capture.Kind {
		case ast.LocalCapture:
			a.resolveLocalCapture(capture, current, lambdaScope, flow, lambdaFlow, callable)
		case ast.GlobalCapture:
			a.resolveGlobalDependency(capture, current, lambdaScope, flow, lambdaFlow, callable)
		}
	}
}

// resolveLocalCapture resolves one `#name`/`Local name` entry: it must name
// an enclosing lexical binding, captured by value into a read-only binding of
// the same static type. A dependency written with the wrong sigil (`#name`
// naming an actual module binding) is still installed under its real kind
// after the diagnostic, so a body reference to the same name does not also
// report a separate "missing dependency" error.
func (a *analyzer) resolveLocalCapture(capture *ast.CaptureRef, current, lambdaScope *scope, flow, lambdaFlow flowState, callable *Callable) {
	outer, owner := current.lookup(capture.Name)
	if outer == nil {
		a.error(codeUnknownCapture, fmt.Sprintf("unknown lambda dependency %q", capture.Name),
			capture.Span(), "capture a binding that is visible where the lambda is written")
		return
	}
	if owner == a.module {
		if outer.Builtin || outer.Kind == ClassSymbol || outer.Kind == FunctionSymbol || outer.Kind == NamespaceSymbol {
			a.error(codeInvalidCapture, fmt.Sprintf("%q is reached directly and does not need a lambda dependency", capture.Name),
				capture.Span(), fmt.Sprintf("remove %q from the lambda dependency list", capture.Name))
			return
		}
		a.error(codeInvalidCapture, fmt.Sprintf("%q is a module binding, not a local one", capture.Name),
			capture.Span(), fmt.Sprintf("write @%s or Global %s in the lambda dependency list", capture.Name, capture.Name))
		a.installGlobalAlias(capture, outer, lambdaScope, flow, lambdaFlow)
		return
	}
	if !capturableSymbol(outer, owner, a.module) {
		a.error(codeInvalidCapture,
			fmt.Sprintf("%q is not a capturable lexical binding", capture.Name),
			capture.Span(), captureRejectionHint(outer, owner, a.module))
		return
	}
	a.installLocalCapture(capture, outer, lambdaScope, flow, lambdaFlow, callable)
}

// resolveGlobalDependency resolves one `@name`/`Global name` entry: it must
// name a module-root value binding, and installs an alias to that binding
// directly in the lambda's scope -- the same alias machinery an ordinary
// Function's `x: Global Type` declaration uses, so the lambda observes the
// real module binding under the existing global-mutation rules rather than a
// closure snapshot. A dependency written with the wrong sigil (`@name` naming
// an actual enclosing local) is still installed under its real kind after the
// diagnostic, for the same no-cascade reason as resolveLocalCapture.
func (a *analyzer) resolveGlobalDependency(capture *ast.CaptureRef, current, lambdaScope *scope, flow, lambdaFlow flowState, callable *Callable) {
	if moduleSymbol, ok := a.module.local(capture.Name); ok {
		if moduleSymbol.Builtin || moduleSymbol.Kind == ClassSymbol || moduleSymbol.Kind == FunctionSymbol || moduleSymbol.Kind == NamespaceSymbol {
			a.error(codeInvalidCapture,
				fmt.Sprintf("%q is reached directly and does not need a Global dependency", capture.Name),
				capture.Span(), fmt.Sprintf("remove %q from the lambda dependency list", capture.Name))
			return
		}
		a.installGlobalAlias(capture, moduleSymbol, lambdaScope, flow, lambdaFlow)
		return
	}
	if outer, owner := current.lookup(capture.Name); outer != nil && owner != a.module && isLexicalCapture(outer.Kind) {
		a.error(codeInvalidCapture, fmt.Sprintf("%q is a local binding, not a module binding", capture.Name),
			capture.Span(), fmt.Sprintf("write #%s or Local %s in the lambda dependency list", capture.Name, capture.Name))
		if capturableSymbol(outer, owner, a.module) {
			a.installLocalCapture(capture, outer, lambdaScope, flow, lambdaFlow, callable)
		}
		return
	}
	a.error(codeUnknownCapture, fmt.Sprintf("no module-root value binding named %q", capture.Name),
		capture.Span(), "declare the module binding before this callable, or remove it from the lambda dependency list")
}

// installLocalCapture records one resolved Local capture: a read-only binding
// inside the lambda's scope, holding the enclosing binding's value at the
// moment the lambda is created.
func (a *analyzer) installLocalCapture(capture *ast.CaptureRef, outer *Symbol, lambdaScope *scope, flow, lambdaFlow flowState, callable *Callable) {
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

// installGlobalAlias records one resolved Global dependency: an alias to the
// real module binding, installed directly in the lambda's scope exactly like
// an ordinary Function's Global declaration installs one in the Function's
// scope.
func (a *analyzer) installGlobalAlias(capture *ast.CaptureRef, moduleSymbol *Symbol, lambdaScope *scope, flow, lambdaFlow flowState) {
	alias := &Symbol{
		Name: capture.Name, Kind: BindingSymbol, Type: moduleSymbol.Type,
		Span: capture.Span(), Declaration: capture, Constant: moduleSymbol.Constant,
		InitialNull: flow.state(moduleSymbol), Alias: moduleSymbol, DeclaredNullable: moduleSymbol.DeclaredNullable,
	}
	lambdaScope.symbols[capture.Name] = alias
	lambdaFlow[alias] = flow.state(moduleSymbol)
	a.result.Symbols = append(a.result.Symbols, alias)
	a.result.ResolvedSymbols[capture] = alias
}

// capturableSymbol reports whether one resolved name may be captured. Only a
// binding belonging to an enclosing callable qualifies: a module-root name is
// reached through an explicit Global dependency instead, and a Class,
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
		return fmt.Sprintf("%q is reachable by ordinary lookup, so remove it from the lambda dependency list", symbol.Name)
	}
	if symbol.Alias != nil {
		return fmt.Sprintf("%q is itself a Global alias; write @%s (or Global %s) to reach the module binding directly", symbol.Name, symbol.Name, symbol.Name)
	}
	switch symbol.Kind {
	case ClassSymbol, NamespaceSymbol, FunctionSymbol:
		return fmt.Sprintf("%q is a declaration rather than a value binding; use it directly", symbol.Name)
	default:
		return "capture an enclosing Function parameter or Local binding"
	}
}

// captureBinding reports the capture entry that already covers one name
// inside a lambda, so reading it is accepted and reading an unlisted
// enclosing local is not.
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
