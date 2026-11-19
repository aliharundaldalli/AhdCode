package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

const TemplatesModuleID = "builtin:Templates"

const templateDataClassID = ir.ClassID(TemplatesModuleID + "::class::TemplateData")
const templateSetClassID = ir.ClassID(TemplatesModuleID + "::class::TemplateSet")
const templateErrorClassID = ir.ClassID(TemplatesModuleID + "::class::TemplateError")

var TemplateDataTextFieldID = ir.FieldID(string(templateDataClassID) + "::field::text")
var TemplateSetTextFieldID = ir.FieldID(string(templateSetClassID) + "::field::text")

func templatesModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	module.Classes = append(module.Classes, templateValueClass(templateDataClassID, "TemplateData", nil))
	module.Functions = append(module.Functions, templateValueConstructor(module.Classes[0]))
	module.Classes = append(module.Classes, templateValueClass(templateSetClassID, "TemplateSet", semantic.TemplateSetOperations))
	module.Functions = append(module.Functions, templateValueConstructor(module.Classes[1]))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: templateErrorClassID, Symbol: ir.SymbolID(string(templateErrorClassID) + "::symbol"),
		Name: "TemplateError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(templateErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

func templateValueClass(id ir.ClassID, name string, operations []string) *ir.Class {
	return &ir.Class{
		ID: id, Symbol: ir.SymbolID(string(id) + "::symbol"),
		Name: name, Operations: operations,
		Fields: []ir.Field{{
			ID: ir.FieldID(string(id) + "::field::text"), Name: "text",
			Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true,
		}},
		Constructor: ir.CallableID(string(id) + "::constructor::(text:String)->Nothing"),
	}
}

func templateValueConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	field := class.Fields[0]
	parameter := ir.Parameter{
		ID: ir.SymbolID(string(id) + "::parameter::text"), Name: "text",
		Type: field.Type, NullState: ir.NonNull,
	}
	return &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature: ir.Signature{
			Parameters: []ir.ParameterType{{Name: "text", Type: field.Type}},
			Return:     ir.Type{Kind: ir.NothingType},
		},
		ReturnNull: ir.NonNull,
		Parameters: []ir.Parameter{parameter},
		Body: ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
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
		}}},
	}
}
