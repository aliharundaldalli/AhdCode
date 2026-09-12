package lowering

import "ahdcode/internal/ir"

// CharactersModuleID is the synthetic module that carries the Characters
// standard library's CharactersError Class declaration into the IR. Like Bits,
// it publishes no data-carrying Class — only its error type and plain
// functions over String, Int, and Bool.
const CharactersModuleID = "builtin:Characters"

const charactersErrorClassID = ir.ClassID(CharactersModuleID + "::class::CharactersError")

func charactersModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	class := &ir.Class{
		ID: charactersErrorClassID, Symbol: ir.SymbolID(string(charactersErrorClassID) + "::symbol"),
		Name: "CharactersError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(charactersErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, class)
	module.Functions = append(module.Functions, builtinConstructor(class, parent))
	return module
}
