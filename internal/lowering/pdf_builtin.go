package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// PDFModuleID is the synthetic module that carries the PDF standard library's
// Class declarations into the IR.
const PDFModuleID = "builtin:PDF"

const pdfDocumentClassID = ir.ClassID(PDFModuleID + "::class::PDFDocument")
const pdfErrorClassID = ir.ClassID(PDFModuleID + "::class::PDFError")

// A PDFDocument stores its entire content as one hidden field: an ordered
// List of Strings, each one a private, JSON-encoded content block (heading,
// paragraph, table, image, or page break) -- the same representation Word's
// Document already uses. It is hidden because PDF publishes a PDFDocument
// only through its members (heading(), paragraph(), save(), ...), each of
// which hands back a fresh PDFDocument built from a fresh block list.
var PDFDocumentBlocksFieldID = ir.FieldID(string(pdfDocumentClassID) + "::field::blocks")

func pdfStringList() ir.Type {
	element := ir.Type{Kind: ir.StringType}
	return ir.Type{Kind: ir.ListType, Element: &element}
}

func pdfDocumentFields() []ir.Field {
	return []ir.Field{
		{ID: PDFDocumentBlocksFieldID, Name: "blocks", Type: pdfStringList(), NullState: ir.NonNull, Hidden: true},
	}
}

// pdfModule emits the PDFDocument Class and the PDFError Class as ordinary IR
// classes, mirroring the Word Document/WordError pattern exactly.
func pdfModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	documentClass := &ir.Class{
		ID: pdfDocumentClassID, Symbol: ir.SymbolID(string(pdfDocumentClassID) + "::symbol"),
		Name: "PDFDocument", Operations: semantic.PDFDocumentOperations,
		Fields:      pdfDocumentFields(),
		Constructor: pdfDocumentConstructorID(),
	}
	module.Classes = append(module.Classes, documentClass)
	module.Functions = append(module.Functions, pdfDocumentConstructor(documentClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: pdfErrorClassID, Symbol: ir.SymbolID(string(pdfErrorClassID) + "::symbol"),
		Name: "PDFError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(pdfErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func pdfDocumentConstructorID() ir.CallableID {
	return ir.CallableID(string(pdfDocumentClassID) + "::constructor::(blocks:List<String>)->Nothing")
}

// pdfDocumentConstructor builds the constructor the backend uses to
// materialize one PDFDocument. AhdCode source cannot reach it: the frontend
// publishes PDFDocument without a constructor, so values come only from
// PDF.new(), PDF.fromWord(...), PDF.fromExcel(...), and PDFDocument
// operations, which validate first and copy the storage they are handed.
func pdfDocumentConstructor(class *ir.Class) *ir.Function {
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
