package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// WordModuleID is the synthetic module that carries the Word standard
// library's Class declarations into the IR.
const WordModuleID = "builtin:Word"

const wordDocumentClassID = ir.ClassID(WordModuleID + "::class::Document")
const wordErrorClassID = ir.ClassID(WordModuleID + "::class::WordError")

// A Document stores its entire content as one hidden field: an ordered List
// of Strings, each one a private, JSON-encoded content block (heading,
// paragraph, table, image, or page break). It is hidden because Word
// publishes a Document only through its members (heading(), paragraph(),
// save(), ...), each of which hands back a fresh Document built from a fresh
// block list. Keeping the storage unreadable, and its encoding entirely
// private to the runtime, is what makes a Document immutable and keeps the
// public surface to exactly Document and WordError.
var WordDocumentBlocksFieldID = ir.FieldID(string(wordDocumentClassID) + "::field::blocks")

func wordStringList() ir.Type {
	element := ir.Type{Kind: ir.StringType}
	return ir.Type{Kind: ir.ListType, Element: &element}
}

func wordDocumentFields() []ir.Field {
	return []ir.Field{
		{ID: WordDocumentBlocksFieldID, Name: "blocks", Type: wordStringList(), NullState: ir.NonNull, Hidden: true},
	}
}

// wordModule emits the Document Class and the WordError Class as ordinary IR
// classes, mirroring the Data Table/DataError pattern exactly.
func wordModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	fields := wordDocumentFields()
	documentClass := &ir.Class{
		ID: wordDocumentClassID, Symbol: ir.SymbolID(string(wordDocumentClassID) + "::symbol"),
		Name: "Document", Operations: semantic.WordDocumentOperations,
		Fields:      fields,
		Constructor: wordDocumentConstructorID(),
	}
	module.Classes = append(module.Classes, documentClass)
	module.Functions = append(module.Functions, wordDocumentConstructor(documentClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: wordErrorClassID, Symbol: ir.SymbolID(string(wordErrorClassID) + "::symbol"),
		Name: "WordError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(wordErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func wordDocumentConstructorID() ir.CallableID {
	return ir.CallableID(string(wordDocumentClassID) + "::constructor::(blocks:List<String>)->Nothing")
}

// wordDocumentConstructor builds the constructor the backend uses to
// materialize one Document. AhdCode source cannot reach it: the frontend
// publishes Document without a constructor, so values come only from
// Word.new(), Word.read(...), and Document operations, which validate first
// and copy the storage they are handed.
func wordDocumentConstructor(class *ir.Class) *ir.Function {
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
