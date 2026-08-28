package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// BuiltinModuleID is the synthetic module that carries the language-supplied
// Class catalog into the IR, so a backend never has to hard-code it.
const BuiltinModuleID = "builtin:core"

const (
	builtinObjectClass  = ir.ClassID("builtin:core::class::Object")
	builtinErrorClass   = ir.ClassID("builtin:core::class::Error")
	builtinMessageField = ir.FieldID("builtin:core::class::Error::field::message")
)

// builtinModule emits Object, Error, and the runtime Error subclasses as
// ordinary IR classes. Their identities match the ones the frontend installs,
// so user code that names them resolves to these declarations.
func builtinModule() *ir.Module {
	module := &ir.Module{ID: BuiltinModuleID, Name: "core", SourcePath: BuiltinModuleID}
	object := &ir.Class{
		ID: builtinObjectClass, Symbol: ir.SymbolID(string(builtinObjectClass) + "::symbol"),
		Name: "Object", Builtin: true, Constructor: builtinConstructorID(builtinObjectClass),
	}
	module.Classes = append(module.Classes, object)
	module.Functions = append(module.Functions, builtinConstructor(object, nil))

	errorClass := &ir.Class{
		ID: builtinErrorClass, Symbol: ir.SymbolID(string(builtinErrorClass) + "::symbol"),
		Name: "Error", Parent: builtinObjectClass, Builtin: true,
		Constructor: builtinConstructorID(builtinErrorClass),
		Fields: []ir.Field{{
			ID: builtinMessageField, Name: "message",
			Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull,
		}},
	}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, nil))

	for _, name := range semantic.BuiltinRuntimeErrorNames {
		identity := ir.ClassID("builtin:core::class::" + name)
		class := &ir.Class{
			ID: identity, Symbol: ir.SymbolID(string(identity) + "::symbol"),
			Name: name, Parent: builtinErrorClass, Builtin: true,
			Constructor: builtinConstructorID(identity),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, builtinConstructor(class, errorClass))
	}
	return module
}

func builtinConstructorID(class ir.ClassID) ir.CallableID {
	if class == builtinObjectClass {
		return ir.CallableID(string(class) + "::constructor::()->Nothing")
	}
	return ir.CallableID(string(class) + "::constructor::(message:String)->Nothing")
}

// builtinConstructor mirrors the frontend's built-in construction contract:
// Object takes no arguments and every Error takes one message.
func builtinConstructor(class *ir.Class, parent *ir.Class) *ir.Function {
	id := class.Constructor
	function := &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: ir.SymbolID(string(id) + "::receiver"),
		Signature: ir.Signature{Return: ir.Type{Kind: ir.NothingType}}, ReturnNull: ir.NonNull,
	}
	if class.ID == builtinObjectClass {
		return function
	}
	parameter := ir.Parameter{
		ID: ir.SymbolID(string(id) + "::parameter::message"), Name: "message",
		Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull,
	}
	function.Signature.Parameters = []ir.ParameterType{{Name: "message", Type: parameter.Type}}
	function.Parameters = []ir.Parameter{parameter}
	if parent != nil {
		function.ParentConstructor = parent.Constructor
		function.ParentArguments = []int{0}
		return function
	}
	function.Body = ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
		Target: ir.Target{
			Kind: ir.FieldTarget, Type: parameter.Type, Field: builtinMessageField,
			Receiver: &ir.LoadExpr{
				ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
				Symbol:   function.Receiver,
			},
		},
		Value: &ir.LoadExpr{
			ExprBase: ir.ExprBase{Type: parameter.Type, NullState: ir.NonNull}, Symbol: parameter.ID,
		},
	}}}
	return function
}
