package lowering

import "ahdcode/internal/ir"

const CSVModuleID = "builtin:CSV"

const csvErrorClassID = ir.ClassID(CSVModuleID + "::class::CSVError")

func csvModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: csvErrorClassID, Symbol: ir.SymbolID(string(csvErrorClassID) + "::symbol"),
		Name: "CSVError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(csvErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
