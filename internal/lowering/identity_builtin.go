package lowering

import "ahdcode/internal/ir"

const IdentityModuleID = "builtin:Identity"

const identityErrorClassID = ir.ClassID(IdentityModuleID + "::class::IdentityError")

func identityModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: identityErrorClassID, Symbol: ir.SymbolID(string(identityErrorClassID) + "::symbol"),
		Name: "IdentityError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(identityErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
