package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// DataModuleID is the synthetic module that carries the Data standard
// library's Class declarations into the IR.
const DataModuleID = "builtin:Data"

const dataTableClassID = ir.ClassID(DataModuleID + "::class::Table")
const dataErrorClassID = ir.ClassID(DataModuleID + "::class::DataError")

// A Table stores its schema and cells in two hidden fields: the ordered column
// names, and one List of String cells per row. They are hidden because Data
// publishes a table through its members (columns(), rows(), column(), ...),
// each of which hands back a fresh snapshot. Keeping the storage unreadable is
// what makes a Table immutable: source can neither read the backing Lists nor
// reach them to mutate, and every operation builds a new Table.
var (
	DataTableColumnsFieldID = ir.FieldID(string(dataTableClassID) + "::field::columns")
	DataTableCellsFieldID   = ir.FieldID(string(dataTableClassID) + "::field::cells")
)

func dataStringList() ir.Type {
	element := ir.Type{Kind: ir.StringType}
	return ir.Type{Kind: ir.ListType, Element: &element}
}

func dataCellGrid() ir.Type {
	row := dataStringList()
	return ir.Type{Kind: ir.ListType, Element: &row}
}

// dataTableFields is the Table storage layout. Both fields are Hidden, so
// has / has not report only the published members of DataTableOperations.
func dataTableFields() []ir.Field {
	return []ir.Field{
		{ID: DataTableColumnsFieldID, Name: "columns", Type: dataStringList(), NullState: ir.NonNull, Hidden: true},
		{ID: DataTableCellsFieldID, Name: "cells", Type: dataCellGrid(), NullState: ir.NonNull, Hidden: true},
	}
}

// dataModule emits the Table Class and the DataError Class as ordinary IR
// classes, mirroring the Regex Pattern/RegexError pattern.
func dataModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	fields := dataTableFields()
	tableClass := &ir.Class{
		ID: dataTableClassID, Symbol: ir.SymbolID(string(dataTableClassID) + "::symbol"),
		Name: "Table", Operations: semantic.DataTableOperations,
		Fields:      fields,
		Constructor: dataTableConstructorID(),
	}
	module.Classes = append(module.Classes, tableClass)
	module.Functions = append(module.Functions, dataTableConstructor(tableClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: dataErrorClassID, Symbol: ir.SymbolID(string(dataErrorClassID) + "::symbol"),
		Name: "DataError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(dataErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func dataTableConstructorID() ir.CallableID {
	return ir.CallableID(string(dataTableClassID) +
		"::constructor::(columns:List<String>,cells:List<List<String>>)->Nothing")
}

// dataTableConstructor builds the constructor the backend uses to materialize
// one Table. AhdCode source cannot reach it: the frontend publishes Table
// without a constructor, so values come only from the Data functions and from
// Table operations, which validate first and copy the storage they are handed.
func dataTableConstructor(class *ir.Class) *ir.Function {
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
