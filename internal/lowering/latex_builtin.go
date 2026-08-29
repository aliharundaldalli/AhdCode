package lowering

import "ahdcode/internal/ir"

// LatexModuleID is the compiler-supplied Latex standard module identity.
const LatexModuleID = "builtin:Latex"

const latexErrorClassID = ir.ClassID(LatexModuleID + "::class::LatexError")

// latexModule carries LatexError into IR. Latex functions are direct runtime
// calls and therefore need no synthetic Function body declarations.
func latexModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	class := &ir.Class{
		ID: latexErrorClassID, Symbol: ir.SymbolID(string(latexErrorClassID) + "::symbol"),
		Name: "LatexError", Parent: builtinErrorClass, Builtin: true,
		Constructor: builtinConstructorID(latexErrorClassID),
	}
	parent := &ir.Class{ID: builtinErrorClass, Constructor: builtinConstructorID(builtinErrorClass)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
