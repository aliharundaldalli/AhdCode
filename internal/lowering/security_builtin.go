package lowering

import "ahdcode/internal/ir"

// SecurityModuleID is the synthetic module that carries the Security standard
// library's SecurityError Class declaration into the IR. Like Env and CSV, it
// publishes no data-carrying Class — only its error type and plain functions
// that return String/Bool/Nothing.
const SecurityModuleID = "builtin:Security"

const securityErrorClassID = ir.ClassID(SecurityModuleID + "::class::SecurityError")

func securityModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: securityErrorClassID, Symbol: ir.SymbolID(string(securityErrorClassID) + "::symbol"),
		Name: "SecurityError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(securityErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
