package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const templatesModuleID = "builtin:Templates"

var (
	templateErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	templateErrorClass = &types.ClassSymbol{ModuleID: templatesModuleID, Name: "TemplateError", Parent: templateErrorParent}
	templateDataClass  = &types.ClassSymbol{ModuleID: templatesModuleID, Name: "TemplateData"}
	templateSetClass   = &types.ClassSymbol{ModuleID: templatesModuleID, Name: "TemplateSet"}
)

func TemplateErrorIdentity() *types.ClassSymbol { return templateErrorClass }
func TemplateDataIdentity() *types.ClassSymbol  { return templateDataClass }
func TemplateSetIdentity() *types.ClassSymbol   { return templateSetClass }

// TemplateSetOperations are the members a loaded template set publishes.
// TemplateData is opaque: values are built with Templates.from* and passed
// to render or layout, and they have no members of their own.
var TemplateSetOperations = []string{"render", "layout"}

func templateDataType() types.Type { return types.Class{Symbol: templateDataClass} }
func templateSetType() types.Type  { return types.Class{Symbol: templateSetClass} }

func templatesModuleInterface() *ModuleInterface {
	module := standardInterface(templatesModuleID, "Templates")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
		error    bool
	}{
		{"TemplateError", templateErrorClass, true},
		{"TemplateData", templateDataClass, false},
		{"TemplateSet", templateSetClass, false},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: templatesModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.error {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[templatesModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	data := templateDataType()
	set := templateSetType()
	names := types.List{Element: types.String}
	values := types.List{Element: data}
	object := types.Pair{Key: types.String, Value: data}
	addStandardExport(module, standardFunction(templatesModuleID, "nullValue", data))
	addStandardExport(module, standardFunction(templatesModuleID, "fromBool", data, types.Parameter{Name: "value", Type: types.Bool}))
	addStandardExport(module, standardFunction(templatesModuleID, "fromInt", data, types.Parameter{Name: "value", Type: types.Int}))
	addStandardExport(module, standardFunction(templatesModuleID, "fromReal", data, types.Parameter{Name: "value", Type: types.Real}))
	addStandardExport(module, standardFunction(templatesModuleID, "fromString", data, types.Parameter{Name: "value", Type: types.String}))
	addStandardExport(module, standardFunction(templatesModuleID, "list", data, types.Parameter{Name: "values", Type: values}))
	addStandardExport(module, standardFunction(templatesModuleID, "object", data, types.Parameter{Name: "values", Type: object}))
	addStandardExport(module, standardFunction(templatesModuleID, "trusted", data, types.Parameter{Name: "markup", Type: types.String}))
	addStandardExport(module, standardFunction(templatesModuleID, "parse", set,
		types.Parameter{Name: "name", Type: types.String},
		types.Parameter{Name: "source", Type: types.String}))
	addStandardExport(module, standardFunction(templatesModuleID, "sources", set,
		types.Parameter{Name: "names", Type: names},
		types.Parameter{Name: "bodies", Type: names}))
	addStandardExport(module, standardFunction(templatesModuleID, "files", set,
		types.Parameter{Name: "root", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

func templateConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != templatesModuleID {
		return "", false
	}
	switch identity.Name {
	case "TemplateData":
		return "create TemplateData with Templates.nullValue, fromBool, fromInt, fromReal, fromString, list, object, or trusted", true
	case "TemplateSet":
		return "create a TemplateSet with Templates.parse, Templates.sources, or Templates.files", true
	}
	return "", false
}

type templateOperationShape struct {
	parameters []types.Type
	result     types.Type
	hint       string
}

func templateOperationShapes() map[TypeOperation]templateOperationShape {
	data := templateDataType()
	return map[TypeOperation]templateOperationShape{
		TemplateSetRender: {[]types.Type{types.String, data}, types.String, "pass a template name and TemplateData"},
		TemplateSetLayout: {[]types.Type{types.String, types.String, data}, types.String, "pass a layout name, a page name, and TemplateData"},
	}
}

var templateOperationNames = map[string]TypeOperation{
	"render": TemplateSetRender,
	"layout": TemplateSetLayout,
}

func templateOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil ||
		class.Symbol.ModuleID != templatesModuleID || class.Symbol.Name != "TemplateSet" {
		return "", false
	}
	operation, known := templateOperationNames[name]
	return operation, known
}

func (a *analyzer) analyzeTemplateOperation(call *ast.CallExpr, operation TypeOperation, shape templateOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, expected := range shape.parameters {
		argument := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
		if argument.invalid() {
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[index].Value, argument.nullState)
			continue
		}
		if !types.Assignable(expected, argument.typeValue) {
			a.typeMismatch(call.Arguments[index].Span(), expected, argument.typeValue, string(operation)+" argument")
		}
	}
	return result
}
