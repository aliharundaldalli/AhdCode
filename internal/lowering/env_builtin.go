package lowering

import "ahdcode/internal/ir"

// EnvModuleID is the synthetic module that carries the Env standard
// library's EnvError Class declaration into the IR. Env, unlike Word/JSON/
// XML, publishes no data-carrying Class - it is a thin wrapper over
// process-environment and .env-file operations - so its only IR class is
// its error type, the same shape CSV uses for CSVError.
const EnvModuleID = "builtin:Env"

const envErrorClassID = ir.ClassID(EnvModuleID + "::class::EnvError")

func envModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: envErrorClassID, Symbol: ir.SymbolID(string(envErrorClassID) + "::symbol"),
		Name: "EnvError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(envErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
