package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const htmlModuleID = "builtin:HTML"

var (
	htmlErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	htmlErrorClass    = &types.ClassSymbol{ModuleID: htmlModuleID, Name: "HTMLError", Parent: htmlErrorParent}
	htmlNodeClass     = &types.ClassSymbol{ModuleID: htmlModuleID, Name: "HTMLNode"}
	htmlDocumentClass = &types.ClassSymbol{ModuleID: htmlModuleID, Name: "HTMLDocument"}
	htmlElementClass  = &types.ClassSymbol{ModuleID: htmlModuleID, Name: "HTMLElement"}
)

// HTMLErrorIdentity, HTMLNodeIdentity, HTMLDocumentIdentity, and
// HTMLElementIdentity expose the canonical identities to lowering without
// coupling the public module interface to a backend.
func HTMLErrorIdentity() *types.ClassSymbol    { return htmlErrorClass }
func HTMLNodeIdentity() *types.ClassSymbol     { return htmlNodeClass }
func HTMLDocumentIdentity() *types.ClassSymbol { return htmlDocumentClass }
func HTMLElementIdentity() *types.ClassSymbol  { return htmlElementClass }

// HTMLNodeOperations is empty: HTMLNode is an immutable builder value with no
// instance members in v0.4.0. The slice exists so has/has not and the IR Class
// stay in agreement with the frontend.
var HTMLNodeOperations = []string{}

// HTMLDocumentOperations and HTMLElementOperations name the members each
// parsed Class publishes (v0.7.0), so has/has not and the IR Class agree
// with the frontend. HTMLDocument/HTMLElement are read-only: there is no
// mutation operation in either list.
var HTMLDocumentOperations = []string{"select", "first"}
var HTMLElementOperations = []string{"tag", "text", "attr", "hasAttr", "select", "first"}

func htmlNodeType() types.Type     { return types.Class{Symbol: htmlNodeClass} }
func htmlDocumentType() types.Type { return types.Class{Symbol: htmlDocumentClass} }
func htmlElementType() types.Type  { return types.Class{Symbol: htmlElementClass} }

func htmlModuleInterface() *ModuleInterface {
	module := standardInterface(htmlModuleID, "HTML")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"HTMLError", htmlErrorClass}, {"HTMLNode", htmlNodeClass},
		{"HTMLDocument", htmlDocumentClass}, {"HTMLElement", htmlElementClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: htmlModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "HTMLError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[htmlModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	node := htmlNodeType()
	attributes := types.Pair{Key: types.String, Value: types.String}
	children := types.List{Element: node}
	addStandardExport(module, standardFunction(htmlModuleID, "text", node,
		types.Parameter{Name: "value", Type: types.String}))
	addStandardExport(module, standardFunction(htmlModuleID, "element", node,
		types.Parameter{Name: "name", Type: types.String},
		types.Parameter{Name: "attributes", Type: attributes},
		types.Parameter{Name: "children", Type: children}))
	addStandardExport(module, standardFunction(htmlModuleID, "render", types.String,
		types.Parameter{Name: "node", Type: node}))
	addStandardExport(module, standardFunction(htmlModuleID, "document", types.String,
		types.Parameter{Name: "title", Type: types.String},
		types.Parameter{Name: "body", Type: children}))
	addStandardExport(module, standardFunction(htmlModuleID, "parse", htmlDocumentType(),
		types.Parameter{Name: "source", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

func htmlConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != htmlModuleID {
		return "", false
	}
	switch identity.Name {
	case "HTMLNode":
		return "create an HTMLNode with HTML.text(value) or HTML.element(name, attributes, children)", true
	case "HTMLDocument":
		return "create an HTMLDocument with HTML.parse(source)", true
	case "HTMLElement":
		return "HTMLElement values are produced by HTMLDocument.select, HTMLDocument.first, HTMLElement.select, or HTMLElement.first", true
	}
	return "", false
}

type htmlOperationShape struct {
	parameters     []types.Type
	result         types.Type
	resultNullable bool
	hint           string
}

func htmlOperationShapes() map[TypeOperation]htmlOperationShape {
	none := []types.Type{}
	elements := types.List{Element: htmlElementType()}
	return map[TypeOperation]htmlOperationShape{
		HTMLDocumentSelect: {[]types.Type{types.String}, elements, false, "pass one String CSS-like selector"},
		HTMLDocumentFirst:  {[]types.Type{types.String}, htmlElementType(), true, "pass one String CSS-like selector"},
		HTMLElementTag:     {none, types.String, false, "call tag with no argument"},
		HTMLElementText:    {none, types.String, false, "call text with no argument"},
		HTMLElementAttr:    {[]types.Type{types.String}, types.String, true, "pass one String attribute name"},
		HTMLElementHasAttr: {[]types.Type{types.String}, types.Bool, false, "pass one String attribute name"},
		HTMLElementSelect:  {[]types.Type{types.String}, elements, false, "pass one String CSS-like selector"},
		HTMLElementFirst:   {[]types.Type{types.String}, htmlElementType(), true, "pass one String CSS-like selector"},
	}
}

var htmlOperationNames = map[string]map[string]TypeOperation{
	"HTMLDocument": {"select": HTMLDocumentSelect, "first": HTMLDocumentFirst},
	"HTMLElement": {
		"tag": HTMLElementTag, "text": HTMLElementText,
		"attr": HTMLElementAttr, "hasAttr": HTMLElementHasAttr,
		"select": HTMLElementSelect, "first": HTMLElementFirst,
	},
}

func htmlOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != htmlModuleID {
		return "", false
	}
	operation, known := htmlOperationNames[class.Symbol.Name][name]
	return operation, known
}

func (a *analyzer) analyzeHTMLOperation(call *ast.CallExpr, operation TypeOperation, shape htmlOperationShape, current *scope, flow flowState) expressionInfo {
	nullState := NonNull
	if shape.resultNullable {
		nullState = MaybeNull
	}
	result := expressionInfo{typeValue: shape.result, nullState: nullState}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, argument := range call.Arguments {
		expected := shape.parameters[index]
		info := a.analyzeExpressionExpected(argument.Value, current, flow, expected)
		if info.invalid() {
			continue
		}
		if info.nullState != NonNull {
			a.nullableError(string(operation), argument.Value, info.nullState)
			continue
		}
		if !types.Assignable(expected, info.typeValue) {
			a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" argument")
		}
	}
	parameters := make([]types.Parameter, len(shape.parameters))
	for index, expected := range shape.parameters {
		parameters[index] = types.Parameter{Type: expected}
	}
	a.result.SelectedCallables[call] = &Callable{
		Signature:  &types.Signature{Parameters: parameters, Return: shape.result},
		ReturnNull: nullState,
	}
	return result
}
