package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const xmlModuleID = "builtin:XML"

var (
	xmlErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	xmlErrorClass    = &types.ClassSymbol{ModuleID: xmlModuleID, Name: "XMLError", Parent: xmlErrorParent}
	xmlNodeClass     = &types.ClassSymbol{ModuleID: xmlModuleID, Name: "XMLNode"}
	xmlDocumentClass = &types.ClassSymbol{ModuleID: xmlModuleID, Name: "XMLDocument"}
)

// XMLErrorIdentity, XMLNodeIdentity, and XMLDocumentIdentity expose the
// canonical identities to the lowering layer without coupling the public
// module interface to a backend.
func XMLErrorIdentity() *types.ClassSymbol    { return xmlErrorClass }
func XMLNodeIdentity() *types.ClassSymbol     { return xmlNodeClass }
func XMLDocumentIdentity() *types.ClassSymbol { return xmlDocumentClass }

// XMLNodeOperations and XMLDocumentOperations name the members each Class
// publishes through built-in type operations, so has/has not reports what a
// value really offers, and the lowering layer's IR Classes stay in
// agreement with the frontend about the published surface.
var XMLNodeOperations = []string{
	"kind", "name", "namespace", "text", "attribute", "attributes", "children", "elements",
}
var XMLDocumentOperations = []string{"root"}

func xmlNodeType() types.Type     { return types.Class{Symbol: xmlNodeClass} }
func xmlDocumentType() types.Type { return types.Class{Symbol: xmlDocumentClass} }

func xmlModuleInterface() *ModuleInterface {
	module := standardInterface(xmlModuleID, "XML")

	errorSymbol := &Symbol{
		Name: "XMLError", Kind: ClassSymbol, Class: xmlErrorClass,
		Type: types.Class{Symbol: xmlErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: xmlModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[xmlModuleID+"\x00XMLError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	nodeSymbol := &Symbol{
		Name: "XMLNode", Kind: ClassSymbol, Class: xmlNodeClass,
		Type: types.Class{Symbol: xmlNodeClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: xmlModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[xmlModuleID+"\x00XMLNode"] = nodeSymbol
	addStandardExport(module, nodeSymbol)

	documentSymbol := &Symbol{
		Name: "XMLDocument", Kind: ClassSymbol, Class: xmlDocumentClass,
		Type: types.Class{Symbol: xmlDocumentClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: xmlModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[xmlModuleID+"\x00XMLDocument"] = documentSymbol
	addStandardExport(module, documentSymbol)

	node := xmlNodeType()
	document := xmlDocumentType()
	attributes := types.Pair{Key: types.String, Value: types.String}
	children := types.List{Element: node}
	stringParameter := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	pretty := types.Parameter{Name: "pretty", Type: types.Bool, HasDefault: true}

	addStandardExport(module, standardFunction(xmlModuleID, "text", node, stringParameter("value")))
	addStandardExport(module, standardFunction(xmlModuleID, "element", node,
		stringParameter("name"),
		types.Parameter{Name: "attributes", Type: attributes},
		types.Parameter{Name: "children", Type: children}))
	addStandardExport(module, standardFunction(xmlModuleID, "document", document,
		types.Parameter{Name: "root", Type: node}))
	addStandardExport(module, standardFunction(xmlModuleID, "parse", document, stringParameter("source")))
	addStandardExport(module, standardFunction(xmlModuleID, "read", document, stringParameter("path")))
	addStandardExport(module, standardFunction(xmlModuleID, "stringify", types.String,
		types.Parameter{Name: "document", Type: document}, pretty))
	addStandardExport(module, standardFunction(xmlModuleID, "write", types.Nothing,
		types.Parameter{Name: "document", Type: document}, stringParameter("path"), pretty))

	sort.Strings(module.ExportNames)
	return module
}

// xmlConstructionHint names the XML functions that produce an XMLNode or
// XMLDocument, so direct construction has an actionable message instead of
// a generic missing-constructor diagnostic.
func xmlConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != xmlModuleID {
		return "", false
	}
	switch identity.Name {
	case "XMLNode":
		return "create an XMLNode with XML.text(value) or XML.element(name, attributes, children)", true
	case "XMLDocument":
		return "create an XMLDocument with XML.document(root), XML.parse(source), or XML.read(path)", true
	}
	return "", false
}

// xmlOperationShape is the fixed call shape of one XMLNode/XMLDocument
// member.
type xmlOperationShape struct {
	receiver   *types.ClassSymbol
	parameters []types.Type
	result     types.Type
	nullable   bool
	hint       string
}

func xmlOperationShapes() map[TypeOperation]xmlOperationShape {
	none := []types.Type{}
	node := xmlNodeType()
	return map[TypeOperation]xmlOperationShape{
		XMLNodeKind:       {xmlNodeClass, none, types.String, false, "call kind with no argument"},
		XMLNodeName:       {xmlNodeClass, none, types.String, false, "call name with no argument"},
		XMLNodeNamespace:  {xmlNodeClass, none, types.String, false, "call namespace with no argument"},
		XMLNodeText:       {xmlNodeClass, none, types.String, false, "call text with no argument"},
		XMLNodeAttribute:  {xmlNodeClass, []types.Type{types.String}, types.String, true, "pass one String attribute name"},
		XMLNodeAttributes: {xmlNodeClass, none, types.Pair{Key: types.String, Value: types.String}, false, "call attributes with no argument"},
		XMLNodeChildren:   {xmlNodeClass, none, types.List{Element: node}, false, "call children with no argument"},
		XMLNodeElements:   {xmlNodeClass, none, types.List{Element: node}, false, "call elements with no argument"},
		XMLDocumentRoot:   {xmlDocumentClass, none, node, false, "call root with no argument"},
	}
}

var xmlOperationNames = map[string]TypeOperation{
	"kind": XMLNodeKind, "name": XMLNodeName, "namespace": XMLNodeNamespace,
	"text": XMLNodeText, "attribute": XMLNodeAttribute, "attributes": XMLNodeAttributes,
	"children": XMLNodeChildren, "elements": XMLNodeElements,
}
var xmlDocumentOperationNames = map[string]TypeOperation{"root": XMLDocumentRoot}

// xmlOperationFor names the built-in member an XMLNode or XMLDocument
// instance publishes. Only the compiler-supplied identities match, so a
// user Class with the same name never collides with either.
func xmlOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != xmlModuleID {
		return "", false
	}
	switch class.Symbol.Name {
	case "XMLNode":
		operation, known := xmlOperationNames[name]
		return operation, known
	case "XMLDocument":
		operation, known := xmlDocumentOperationNames[name]
		return operation, known
	}
	return "", false
}

// analyzeXMLOperation checks one XMLNode/XMLDocument member, following the
// same shape convention analyzeJSONOperation and analyzeRegexOperation use:
// every argument is a NonNull value of the declared type, and only
// attribute()'s result is statically MaybeNull.
func (a *analyzer) analyzeXMLOperation(call *ast.CallExpr, operation TypeOperation, shape xmlOperationShape, current *scope, flow flowState) expressionInfo {
	nullState := NonNull
	if shape.nullable {
		nullState = MaybeNull
	}
	result := expressionInfo{typeValue: shape.result, nullState: nullState}
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
