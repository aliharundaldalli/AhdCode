package lowering

import "ahdcode/internal/ir"

// ArchiveModuleID is the synthetic module that carries the Archive standard
// library's Class declarations into the IR. Archive is creation-only and
// publishes no data Class of its own -- only ArchiveError needs IR presence,
// exactly like File/FileError.
const ArchiveModuleID = "builtin:Archive"

const archiveErrorClassID = ir.ClassID(ArchiveModuleID + "::class::ArchiveError")

func archiveModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: archiveErrorClassID, Symbol: ir.SymbolID(string(archiveErrorClassID) + "::symbol"),
		Name: "ArchiveError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(archiveErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
