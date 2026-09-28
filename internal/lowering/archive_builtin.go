package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// ArchiveModuleID is the synthetic module that carries the Archive standard
// library's Class declarations into the IR: ArchiveError and, since v2.5.0,
// the ArchiveEntry value Archive.list returns.
const ArchiveModuleID = "builtin:Archive"

const (
	archiveErrorClassID = ir.ClassID(ArchiveModuleID + "::class::ArchiveError")
	archiveEntryClassID = ir.ClassID(ArchiveModuleID + "::class::ArchiveEntry")
)

// ArchiveEntryDataFieldID is the hidden String holding one entry's encoded
// path, kind, and size.
var ArchiveEntryDataFieldID = ir.FieldID(string(archiveEntryClassID) + "::field::data")

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
	addHiddenStringValueClass(module, archiveEntryClassID, "ArchiveEntry", ArchiveEntryDataFieldID, semantic.ArchiveEntryOperations)
	return module
}
