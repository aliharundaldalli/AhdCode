package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

const ExcelModuleID = "builtin:Excel"

const (
	excelWorkbookClassID = ir.ClassID(ExcelModuleID + "::class::Workbook")
	excelSheetClassID    = ir.ClassID(ExcelModuleID + "::class::Sheet")
	excelCellClassID     = ir.ClassID(ExcelModuleID + "::class::Cell")
	excelRangeClassID    = ir.ClassID(ExcelModuleID + "::class::Range")
	excelStyleClassID    = ir.ClassID(ExcelModuleID + "::class::CellStyle")
	excelErrorClassID    = ir.ClassID(ExcelModuleID + "::class::ExcelError")
)

var (
	ExcelWorkbookDataFieldID = ir.FieldID(string(excelWorkbookClassID) + "::field::data")
	ExcelSheetDataFieldID    = ir.FieldID(string(excelSheetClassID) + "::field::data")
	ExcelCellDataFieldID     = ir.FieldID(string(excelCellClassID) + "::field::data")
	ExcelRangeDataFieldID    = ir.FieldID(string(excelRangeClassID) + "::field::data")
	ExcelStyleDataFieldID    = ir.FieldID(string(excelStyleClassID) + "::field::data")
)

type excelClassSpec struct {
	id         ir.ClassID
	name       string
	field      ir.FieldID
	operations []string
}

func excelModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []excelClassSpec{
		{excelWorkbookClassID, "Workbook", ExcelWorkbookDataFieldID, semantic.ExcelWorkbookOperations},
		{excelSheetClassID, "Sheet", ExcelSheetDataFieldID, semantic.ExcelSheetOperations},
		{excelCellClassID, "Cell", ExcelCellDataFieldID, semantic.ExcelCellOperations},
		{excelRangeClassID, "Range", ExcelRangeDataFieldID, semantic.ExcelRangeOperations},
		{excelStyleClassID, "CellStyle", ExcelStyleDataFieldID, semantic.ExcelStyleOperations},
	}
	for _, spec := range specs {
		field := ir.Field{ID: spec.field, Name: "data", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"), Name: spec.name,
			Operations: spec.operations, Fields: []ir.Field{field}, Constructor: excelValueConstructorID(spec.id),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, excelValueConstructor(class))
	}

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: excelErrorClassID, Symbol: ir.SymbolID(string(excelErrorClassID) + "::symbol"),
		Name: "ExcelError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(excelErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

func excelValueConstructorID(class ir.ClassID) ir.CallableID {
	return ir.CallableID(string(class) + "::constructor::(data:String)->Nothing")
}

func excelValueConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	field := class.Fields[0]
	parameter := ir.Parameter{
		ID: ir.SymbolID(string(id) + "::parameter::data"), Name: "data",
		Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull,
	}
	return &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature:  ir.Signature{Parameters: []ir.ParameterType{{Name: "data", Type: parameter.Type}}, Return: ir.Type{Kind: ir.NothingType}},
		Parameters: []ir.Parameter{parameter}, ReturnNull: ir.NonNull,
		Body: ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
			Target: ir.Target{Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull}, Symbol: receiver}},
			Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID},
		}}},
	}
}
