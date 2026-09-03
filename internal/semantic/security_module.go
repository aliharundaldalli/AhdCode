package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const securityModuleID = "builtin:Security"

var securityErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
var securityErrorClass = &types.ClassSymbol{
	ModuleID: securityModuleID, Name: "SecurityError",
	Parent: securityErrorParent,
}

// SecurityErrorIdentity exposes the canonical identity to the lowering layer
// without coupling the public module interface to a backend.
func SecurityErrorIdentity() *types.ClassSymbol { return securityErrorClass }

func securityModuleInterface() *ModuleInterface {
	module := standardInterface(securityModuleID, "Security")
	errorSymbol := &Symbol{
		Name: "SecurityError", Kind: ClassSymbol, Class: securityErrorClass,
		Type: types.Class{Symbol: securityErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: securityModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[securityModuleID+"\x00SecurityError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	name := func(label string) types.Parameter { return types.Parameter{Name: label, Type: types.String} }

	// Security.passwordHash(password: String) -> String
	addStandardExport(module, standardFunction(securityModuleID, "passwordHash", types.String, name("password")))
	// Security.passwordVerify(password: String, encodedHash: String) -> Bool
	addStandardExport(module, standardFunction(securityModuleID, "passwordVerify", types.Bool, name("password"), name("encodedHash")))
	// Security.token() -> String
	addStandardExport(module, standardFunction(securityModuleID, "token", types.String))
	// Security.secureEqual(expected: String, received: String) -> Bool
	addStandardExport(module, standardFunction(securityModuleID, "secureEqual", types.Bool, name("expected"), name("received")))

	sort.Strings(module.ExportNames)
	return module
}
