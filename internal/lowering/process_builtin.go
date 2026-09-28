package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// ProcessModuleID is the compiler-supplied identity of the Process module
// (v2.5.0).
const ProcessModuleID = "builtin:Process"

const (
	processResultClassID = ir.ClassID(ProcessModuleID + "::class::ProcessResult")
	processErrorClassID  = ir.ClassID(ProcessModuleID + "::class::ProcessError")
)

// ProcessResultDataFieldID is the hidden String holding one run's encoded
// exit code, stdout, and stderr.
var ProcessResultDataFieldID = ir.FieldID(string(processResultClassID) + "::field::data")

func processModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	addHiddenStringValueClass(module, processResultClassID, "ProcessResult", ProcessResultDataFieldID, semantic.ProcessResultOperations)
	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: processErrorClassID, Symbol: ir.SymbolID(string(processErrorClassID) + "::symbol"),
		Name: "ProcessError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(processErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

// addHiddenStringValueClass declares one immutable value Class whose single
// hidden String field holds its encoded state -- the representation QRCode,
// UUIDValue, and SMTPClient use. Source code cannot reach the constructor;
// only the owning module's functions produce values.
func addHiddenStringValueClass(module *ir.Module, id ir.ClassID, name string, field ir.FieldID, operations []string) {
	data := ir.Field{ID: field, Name: "data", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
	class := &ir.Class{
		ID: id, Symbol: ir.SymbolID(string(id) + "::symbol"), Name: name,
		Operations: operations, Fields: []ir.Field{data},
		Constructor: ir.CallableID(string(id) + "::constructor::(data:String)->Nothing"),
	}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, smtpValueConstructor(class))
}
