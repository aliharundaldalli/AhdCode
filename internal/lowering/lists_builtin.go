package lowering

import "ahdcode/internal/ir"

// ListsModuleID is the synthetic module that carries the Lists standard
// library's ListsError Class declaration into the IR. Lists publishes no
// data-carrying Class: it transforms the core List type and hands back
// ordinary List values, so its only IR class is its error type, the same
// shape Env and CSV use.
const ListsModuleID = "builtin:Lists"

const listsErrorClassID = ir.ClassID(ListsModuleID + "::class::ListsError")

func listsModule(id ir.ModuleID, name, path string) *ir.Module {
	return collectionErrorModule(id, name, path, listsErrorClassID, "ListsError")
}

// collectionErrorModule emits one standard module whose entire IR contribution
// is a single catchable Error subclass.
func collectionErrorModule(id ir.ModuleID, name, path string, class ir.ClassID, className string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	parentID := ir.ClassID("builtin:core::class::Error")
	declaration := &ir.Class{
		ID: class, Symbol: ir.SymbolID(string(class) + "::symbol"),
		Name: className, Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(class),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, declaration)
	module.Functions = append(module.Functions, builtinConstructor(declaration, parent))
	return module
}
