package lowering

import "ahdcode/internal/ir"

const FileModuleID = "builtin:File"

const fileErrorClassID = ir.ClassID(FileModuleID + "::class::FileError")

func fileModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::IOError")
	class := &ir.Class{
		ID: fileErrorClassID, Symbol: ir.SymbolID(string(fileErrorClassID) + "::symbol"),
		Name: "FileError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(fileErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
