package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const identityModuleID = "builtin:Identity"

var identityErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
var identityErrorClass = &types.ClassSymbol{
	ModuleID: identityModuleID, Name: "IdentityError",
	Parent: identityErrorParent,
}

// IdentityErrorIdentity exposes the canonical identity to the lowering layer.
func IdentityErrorIdentity() *types.ClassSymbol { return identityErrorClass }

func identityModuleInterface() *ModuleInterface {
	module := standardInterface(identityModuleID, "Identity")
	errorSymbol := &Symbol{
		Name: "IdentityError", Kind: ClassSymbol, Class: identityErrorClass,
		Type: types.Class{Symbol: identityErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: identityModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[identityModuleID+"\x00IdentityError"] = errorSymbol
	addStandardExport(module, errorSymbol)
	addStandardExport(module, standardFunction(identityModuleID, "id", types.String))
	sort.Strings(module.ExportNames)
	return module
}
