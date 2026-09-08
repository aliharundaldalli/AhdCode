package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const bitsModuleID = "builtin:Bits"

var bitsErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
var bitsErrorClass = &types.ClassSymbol{
	ModuleID: bitsModuleID, Name: "BitsError",
	Parent: bitsErrorParent,
}

// BitsErrorIdentity exposes the canonical identity to the lowering layer.
func BitsErrorIdentity() *types.ClassSymbol { return bitsErrorClass }

// bitsModuleInterface declares the Bits standard module.
//
// AhdCode's grammar has no bitwise operators, so these operations are named
// calls. Every one is defined on the language's single integer type, signed
// 64-bit Int, over its two's-complement bit pattern.
func bitsModuleInterface() *ModuleInterface {
	module := standardInterface(bitsModuleID, "Bits")
	errorSymbol := &Symbol{
		Name: "BitsError", Kind: ClassSymbol, Class: bitsErrorClass,
		Type: types.Class{Symbol: bitsErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: bitsModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[bitsModuleID+"\x00BitsError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	value := func(label string) types.Parameter { return types.Parameter{Name: label, Type: types.Int} }

	addStandardExport(module, standardFunction(bitsModuleID, "bitAnd", types.Int, value("left"), value("right")))
	addStandardExport(module, standardFunction(bitsModuleID, "bitOr", types.Int, value("left"), value("right")))
	addStandardExport(module, standardFunction(bitsModuleID, "bitXor", types.Int, value("left"), value("right")))
	addStandardExport(module, standardFunction(bitsModuleID, "bitNot", types.Int, value("value")))

	addStandardExport(module, standardFunction(bitsModuleID, "shiftLeft", types.Int, value("value"), value("distance")))
	addStandardExport(module, standardFunction(bitsModuleID, "shiftRight", types.Int, value("value"), value("distance")))
	addStandardExport(module, standardFunction(bitsModuleID, "shiftRightUnsigned", types.Int, value("value"), value("distance")))
	addStandardExport(module, standardFunction(bitsModuleID, "rotateLeft", types.Int, value("value"), value("distance")))
	addStandardExport(module, standardFunction(bitsModuleID, "rotateRight", types.Int, value("value"), value("distance")))

	addStandardExport(module, standardFunction(bitsModuleID, "count", types.Int, value("value")))
	addStandardExport(module, standardFunction(bitsModuleID, "leadingZeros", types.Int, value("value")))
	addStandardExport(module, standardFunction(bitsModuleID, "trailingZeros", types.Int, value("value")))

	sort.Strings(module.ExportNames)
	return module
}
