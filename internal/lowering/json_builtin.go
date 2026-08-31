package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// JSONModuleID is the synthetic module that carries the JSON standard
// library's Class declarations into the IR.
const JSONModuleID = "builtin:JSON"

const jsonValueClassID = ir.ClassID(JSONModuleID + "::class::JSONValue")
const jsonErrorClassID = ir.ClassID(JSONModuleID + "::class::JSONError")

// A JSONValue stores its entire content as one hidden field: its own
// canonical, compact JSON text. It is hidden because JSON publishes a
// JSONValue only through its members (kind(), int(), get(), ...), each of
// which decodes this text, and any member that itself produces a JSONValue
// (array(), get(), at(), ...) hands back a fresh instance built from a fresh
// canonical text - the same "hidden String field, reparsed by helpers"
// pattern Word's Document uses for its block list.
var JSONValueTextFieldID = ir.FieldID(string(jsonValueClassID) + "::field::text")

func jsonValueFields() []ir.Field {
	return []ir.Field{
		{ID: JSONValueTextFieldID, Name: "text", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true},
	}
}

// jsonModule emits the JSONValue Class and the JSONError Class as ordinary IR
// classes, mirroring the Word Document/WordError pattern exactly.
func jsonModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	fields := jsonValueFields()
	valueClass := &ir.Class{
		ID: jsonValueClassID, Symbol: ir.SymbolID(string(jsonValueClassID) + "::symbol"),
		Name: "JSONValue", Operations: semantic.JSONValueOperations,
		Fields:      fields,
		Constructor: jsonValueConstructorID(),
	}
	module.Classes = append(module.Classes, valueClass)
	module.Functions = append(module.Functions, jsonValueConstructor(valueClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: jsonErrorClassID, Symbol: ir.SymbolID(string(jsonErrorClassID) + "::symbol"),
		Name: "JSONError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(jsonErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func jsonValueConstructorID() ir.CallableID {
	return ir.CallableID(string(jsonValueClassID) + "::constructor::(text:String)->Nothing")
}

// jsonValueConstructor builds the constructor the backend uses to
// materialize one JSONValue. AhdCode source cannot reach it: the frontend
// publishes JSONValue without a constructor, so values come only from JSON
// module functions and JSONValue operations, which validate first and
// produce their own canonical text.
func jsonValueConstructor(class *ir.Class) *ir.Function {
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
