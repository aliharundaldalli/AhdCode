package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// GUIModuleID is the synthetic module that carries the GUI standard module's
// Window, Container, Label, Button, TextInput, Checkbox, and GUIError Classes
// into the IR.
const GUIModuleID = "builtin:GUI"

const guiErrorClassID = ir.ClassID(GUIModuleID + "::class::GUIError")

// guiModule emits the GUI Classes as builtin Classes that each hold one
// hidden handle. The window, its widgets, and their values live in the GUI
// runtime and its helper; the handle names them. None of the Classes
// publishes a constructor to AhdCode source: a Window comes from GUI.window
// and every widget from a Window or Container member.
func guiModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	for _, className := range semantic.GUIClassNames {
		classID := ir.ClassID(GUIModuleID + "::class::" + className)
		class := &ir.Class{
			ID: classID, Symbol: ir.SymbolID(string(classID) + "::symbol"),
			Name: className, Operations: semantic.GUIOperations[className],
			Fields: []ir.Field{{
				ID: ir.FieldID(string(classID) + "::field::handle"), Name: "handle",
				Type: ir.Type{Kind: ir.IntType}, NullState: ir.NonNull, Hidden: true,
			}},
			Constructor: plotAllFieldsConstructorID(classID),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, plotAllFieldsConstructor(class))
	}

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: guiErrorClassID, Symbol: ir.SymbolID(string(guiErrorClassID) + "::symbol"),
		Name: "GUIError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(guiErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}
