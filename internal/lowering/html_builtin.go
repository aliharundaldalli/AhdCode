package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

const HTMLModuleID = "builtin:HTML"

const (
	htmlNodeClassID     = ir.ClassID(HTMLModuleID + "::class::HTMLNode")
	htmlDocumentClassID = ir.ClassID(HTMLModuleID + "::class::HTMLDocument")
	htmlElementClassID  = ir.ClassID(HTMLModuleID + "::class::HTMLElement")
	htmlErrorClassID    = ir.ClassID(HTMLModuleID + "::class::HTMLError")
)

var (
	HTMLNodeDataFieldID     = ir.FieldID(string(htmlNodeClassID) + "::field::data")
	HTMLDocumentDataFieldID = ir.FieldID(string(htmlDocumentClassID) + "::field::data")
	HTMLElementDataFieldID  = ir.FieldID(string(htmlElementClassID) + "::field::data")
)

func htmlModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []struct {
		id         ir.ClassID
		name       string
		field      ir.FieldID
		operations []string
	}{
		{htmlNodeClassID, "HTMLNode", HTMLNodeDataFieldID, semantic.HTMLNodeOperations},
		{htmlDocumentClassID, "HTMLDocument", HTMLDocumentDataFieldID, semantic.HTMLDocumentOperations},
		{htmlElementClassID, "HTMLElement", HTMLElementDataFieldID, semantic.HTMLElementOperations},
	}
	for _, spec := range specs {
		field := ir.Field{ID: spec.field, Name: "data", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"),
			Name: spec.name, Operations: spec.operations, Fields: []ir.Field{field},
			Constructor: ir.CallableID(string(spec.id) + "::constructor::(data:String)->Nothing"),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, htmlValueConstructor(class))
	}

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: htmlErrorClassID, Symbol: ir.SymbolID(string(htmlErrorClassID) + "::symbol"),
		Name: "HTMLError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(htmlErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

func htmlValueConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	field := class.Fields[0]
	parameter := ir.Parameter{
		ID: ir.SymbolID(string(id) + "::parameter::" + field.Name), Name: field.Name,
		Type: field.Type, NullState: ir.NonNull,
	}
	return &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature: ir.Signature{
			Parameters: []ir.ParameterType{{Name: field.Name, Type: field.Type}},
			Return:     ir.Type{Kind: ir.NothingType},
		},
		Parameters: []ir.Parameter{parameter}, ReturnNull: ir.NonNull,
		Body: ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
			Target: ir.Target{
				Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{
					ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
					Symbol:   receiver,
				},
			},
			Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID},
		}}},
	}
}
