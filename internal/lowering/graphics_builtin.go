package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// GraphicsModuleID is the synthetic module that carries the Graphics standard
// module's Canvas, Turtle, and GraphicsError Classes into the IR.
const GraphicsModuleID = "builtin:Graphics"

const (
	graphicsCanvasClassID = ir.ClassID(GraphicsModuleID + "::class::Canvas")
	graphicsTurtleClassID = ir.ClassID(GraphicsModuleID + "::class::Turtle")
	graphicsErrorClassID  = ir.ClassID(GraphicsModuleID + "::class::GraphicsError")
)

// graphicsModule emits Canvas and Turtle as builtin Classes that each hold
// one hidden handle. The window, its drawing, and every Turtle's position,
// heading, pen, color, and width live in the Graphics runtime, which both
// the evaluator and compiled programs share; the handle names that state.
// Neither Class publishes a constructor to AhdCode source: a Canvas comes
// from Graphics.open and a Turtle from Canvas.turtle.
func graphicsModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	for _, spec := range []struct {
		id         ir.ClassID
		name       string
		operations []string
	}{
		{graphicsCanvasClassID, "Canvas", semantic.GraphicsCanvasOperations},
		{graphicsTurtleClassID, "Turtle", semantic.GraphicsTurtleOperations},
	} {
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"),
			Name: spec.name, Operations: spec.operations,
			Fields: []ir.Field{{
				ID: ir.FieldID(string(spec.id) + "::field::handle"), Name: "handle",
				Type: ir.Type{Kind: ir.IntType}, NullState: ir.NonNull, Hidden: true,
			}},
			Constructor: plotAllFieldsConstructorID(spec.id),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, plotAllFieldsConstructor(class))
	}

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: graphicsErrorClassID, Symbol: ir.SymbolID(string(graphicsErrorClassID) + "::symbol"),
		Name: "GraphicsError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(graphicsErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}
