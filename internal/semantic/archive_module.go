package semantic

import (
	"ahdcode/internal/types"
)

const archiveModuleID = "builtin:Archive"

var archiveErrorClass = &types.ClassSymbol{
	ModuleID: archiveModuleID, Name: "ArchiveError",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
}

// ArchiveErrorIdentity exposes the standard module's catchable error identity
// to lowering without coupling the public module interface to a backend.
func ArchiveErrorIdentity() *types.ClassSymbol { return archiveErrorClass }

// Archive is creation-only in v0.1.20: no extraction, listing, or archive
// object model. Each function takes a destination path and a Pair mapping an
// in-archive member path (key) to a source filesystem path (value).
func archiveModuleInterface() *ModuleInterface {
	module := standardInterface(archiveModuleID, "Archive")

	errorSymbol := &Symbol{
		Name: "ArchiveError", Kind: ClassSymbol, Class: archiveErrorClass,
		Type: types.Class{Symbol: archiveErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: archiveModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[archiveModuleID+"\x00ArchiveError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	entries := types.Parameter{Name: "entries", Type: types.Pair{Key: types.String, Value: types.String}}
	for _, name := range []string{"zip", "tar", "tarGzip"} {
		addStandardExport(module, standardFunction(archiveModuleID, name, types.Nothing,
			types.Parameter{Name: "output", Type: types.String}, entries))
	}

	return module
}
