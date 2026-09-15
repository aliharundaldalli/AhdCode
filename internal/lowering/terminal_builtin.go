package lowering

import "ahdcode/internal/ir"

// TerminalModuleID is the synthetic module that carries the Terminal standard
// module's TerminalError Class declaration into the IR. Terminal publishes no
// data-carrying Class, so, like Env, its only IR class is its error type.
const TerminalModuleID = "builtin:Terminal"

const terminalErrorClassID = ir.ClassID(TerminalModuleID + "::class::TerminalError")

func terminalModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: terminalErrorClassID, Symbol: ir.SymbolID(string(terminalErrorClassID) + "::symbol"),
		Name: "TerminalError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(terminalErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
