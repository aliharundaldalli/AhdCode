package lowering

import "ahdcode/internal/ir"

// BitsModuleID is the synthetic module that carries the Bits standard
// library's BitsError Class declaration into the IR. Like Security, it
// publishes no data-carrying Class — only its error type and plain functions
// that return Int.
const BitsModuleID = "builtin:Bits"

const bitsErrorClassID = ir.ClassID(BitsModuleID + "::class::BitsError")

func bitsModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: bitsErrorClassID, Symbol: ir.SymbolID(string(bitsErrorClassID) + "::symbol"),
		Name: "BitsError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(bitsErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
