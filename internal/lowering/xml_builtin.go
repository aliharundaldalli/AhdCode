package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// XMLModuleID is the synthetic module that carries the XML standard
// library's Class declarations into the IR.
const XMLModuleID = "builtin:XML"

const xmlNodeClassID = ir.ClassID(XMLModuleID + "::class::XMLNode")
const xmlDocumentClassID = ir.ClassID(XMLModuleID + "::class::XMLDocument")
const xmlErrorClassID = ir.ClassID(XMLModuleID + "::class::XMLError")

// An XMLNode and an XMLDocument each store their entire content as one
// hidden field: an opaque, private encoding of the node tree (Kind, Name,
// Namespace, Text, attributes, and children). Like Word's Document and
// JSON's JSONValue, the encoding is never published - every member (kind(),
// name(), children(), ...) decodes it, and any member that itself produces
// an XMLNode hands back a fresh instance built from a fresh encoding.
var XMLNodeDataFieldID = ir.FieldID(string(xmlNodeClassID) + "::field::data")
var XMLDocumentDataFieldID = ir.FieldID(string(xmlDocumentClassID) + "::field::data")

func xmlDataFields(classID ir.ClassID) []ir.Field {
	return []ir.Field{
		{ID: ir.FieldID(string(classID) + "::field::data"), Name: "data", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true},
	}
}

// xmlModule emits the XMLNode, XMLDocument, and XMLError Classes as
// ordinary IR classes, mirroring the Word Document/WordError pattern.
func xmlModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	nodeFields := xmlDataFields(xmlNodeClassID)
	nodeClass := &ir.Class{
		ID: xmlNodeClassID, Symbol: ir.SymbolID(string(xmlNodeClassID) + "::symbol"),
		Name: "XMLNode", Operations: semantic.XMLNodeOperations,
		Fields:      nodeFields,
		Constructor: xmlConstructorID(xmlNodeClassID),
	}
	module.Classes = append(module.Classes, nodeClass)
	module.Functions = append(module.Functions, xmlDataConstructor(nodeClass))

	documentFields := xmlDataFields(xmlDocumentClassID)
	documentClass := &ir.Class{
		ID: xmlDocumentClassID, Symbol: ir.SymbolID(string(xmlDocumentClassID) + "::symbol"),
		Name: "XMLDocument", Operations: semantic.XMLDocumentOperations,
		Fields:      documentFields,
		Constructor: xmlConstructorID(xmlDocumentClassID),
	}
	module.Classes = append(module.Classes, documentClass)
	module.Functions = append(module.Functions, xmlDataConstructor(documentClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: xmlErrorClassID, Symbol: ir.SymbolID(string(xmlErrorClassID) + "::symbol"),
		Name: "XMLError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(xmlErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func xmlConstructorID(classID ir.ClassID) ir.CallableID {
	return ir.CallableID(string(classID) + "::constructor::(data:String)->Nothing")
}

// xmlDataConstructor builds the constructor the backend uses to materialize
// one XMLNode/XMLDocument. AhdCode source cannot reach it: the frontend
// publishes neither Class with a constructor, so values come only from XML
// module functions and XMLNode/XMLDocument operations.
func xmlDataConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	function := &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature:  ir.Signature{Return: ir.Type{Kind: ir.NothingType}},
		ReturnNull: ir.NonNull,
	}
	var statements []ir.Statement
	for _, field := range class.Fields {
		parameter := ir.Parameter{
			ID: ir.SymbolID(string(id) + "::parameter::" + field.Name), Name: field.Name,
			Type: field.Type, NullState: ir.NonNull,
		}
		function.Signature.Parameters = append(function.Signature.Parameters,
			ir.ParameterType{Name: field.Name, Type: field.Type})
		function.Parameters = append(function.Parameters, parameter)
		statements = append(statements, &ir.AssignStmt{
			Target: ir.Target{
				Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{
					ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
					Symbol:   receiver,
				},
			},
			Value: &ir.LoadExpr{
				ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID,
			},
		})
	}
	function.Body = ir.Block{Statements: statements}
	return function
}
