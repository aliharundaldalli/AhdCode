package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// Terminal (v1.5.0) adds the terminal-specific behaviour the Fundamentals
// write, take, and str deliberately do not expose. It publishes ordinary fixed
// signatures, one catchable TerminalError, and a single type-directed
// operation, pretty, which accepts exactly the values write accepts and lays
// out List and Pair values across lines.

const terminalModuleID = "builtin:Terminal"

var terminalErrorClass = &types.ClassSymbol{
	ModuleID: terminalModuleID, Name: "TerminalError",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
}

// TerminalErrorIdentity exposes the standard module's catchable error identity
// to lowering without coupling the public module interface to a backend.
func TerminalErrorIdentity() *types.ClassSymbol { return terminalErrorClass }

var terminalOperationShapes = map[ModuleOperation]moduleOperationShape{
	TerminalPretty: {1, "pass one value with a textual representation, as write accepts"},
}

func terminalModuleInterface() *ModuleInterface {
	module := standardInterface(terminalModuleID, "Terminal")
	addStandardExport(module, standardErrorClass(terminalModuleID, "TerminalError", terminalErrorClass, module))

	text := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	optionalText := func(name string) types.Parameter {
		return types.Parameter{Name: name, Type: types.String, HasDefault: true}
	}
	optionalFlag := func(name string) types.Parameter {
		return types.Parameter{Name: name, Type: types.Bool, HasDefault: true}
	}

	addStandardExport(module, standardFunction(terminalModuleID, "emit", types.Nothing,
		types.Parameter{Name: "parts", Type: types.List{Element: types.String}},
		optionalText("separator"), optionalText("ending")))
	addStandardExport(module, standardFunction(terminalModuleID, "error", types.Nothing,
		text("text"), optionalText("ending")))
	addStandardExport(module, standardFunction(terminalModuleID, "flush", types.Nothing))
	addStandardExport(module, standardFunction(terminalModuleID, "isInteractive", types.Bool))
	addStandardExport(module, standardNullableFunction(terminalModuleID, "width", types.Int))
	addStandardExport(module, standardNullableFunction(terminalModuleID, "height", types.Int))
	addStandardExport(module, standardFunction(terminalModuleID, "supportsColor", types.Bool))
	addStandardExport(module, standardFunction(terminalModuleID, "style", types.String,
		text("text"), optionalText("foreground"), optionalText("background"),
		optionalFlag("bold"), optionalFlag("underline")))
	addStandardExport(module, moduleOperationSymbol(terminalModuleID, "pretty", TerminalPretty))

	sort.Strings(module.ExportNames)
	return module
}

// analyzeTerminalOperation type-checks Terminal.pretty, reporting false for an
// operation belonging to another module. pretty accepts exactly what write
// accepts, including a value that may be null, and records the argument's own
// static type as the specialized parameter so every later layer lays out the
// value from that one known type.
func (a *analyzer) analyzeTerminalOperation(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) (expressionInfo, bool) {
	if operation != TerminalPretty {
		return expressionInfo{}, false
	}
	info := a.analyzeExpression(call.Arguments[0].Value, current, flow)
	if info.invalid() {
		return moduleOperationFailure(), true
	}
	if !renderable(info.typeValue) {
		a.error(codeCallArguments,
			fmt.Sprintf("%s does not accept %s", operation, types.Display(info.typeValue)),
			call.Arguments[0].Span(), terminalOperationShapes[operation].hint)
		return moduleOperationFailure(), true
	}
	return a.moduleOperationResult(call, operation, types.Nothing,
		[]types.Parameter{{Name: "value", Type: info.typeValue}}, []NullState{info.nullState}), true
}
