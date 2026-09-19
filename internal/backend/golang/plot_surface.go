package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// Plot.surface and the Surface members (v2.0). A Surface is an immutable
// value like a Chart: its hidden fields are read into the runtime's
// AhdSurface, and a member that changes it builds a new value through the
// generated all-fields constructor.

const plotSurfaceClass = ir.ClassID("builtin:Plot::class::Surface")

// plotSurfaceFields lists the Surface storage fields in the order
// internal/lowering/plot_builtin.go declares them, with their AhdSurface
// names.
var plotSurfaceFields = []struct{ field, runtime string }{
	{"x", "X"}, {"y", "Y"}, {"z", "Z"},
	{"title", "Title"}, {"xLabel", "XLabel"}, {"yLabel", "YLabel"}, {"zLabel", "ZLabel"},
	{"width", "Width"}, {"height", "Height"}, {"wireframe", "Wireframe"},
}

func plotSurfaceFieldID(name string) ir.FieldID {
	return ir.FieldID(string(plotSurfaceClass) + "::field::" + name)
}

// plotSurfaceCall lowers Plot.surface(x, y, z).
func (generator *generator) plotSurfaceCall(value *ir.CallExpr, meta ir.ExprBase) string {
	if len(value.Arguments) != 3 || value.Arguments[0].Value == nil || value.Arguments[1].Value == nil || value.Arguments[2].Value == nil {
		generator.fail(CodeGenerationFailure, "Plot.surface has a missing argument", meta.Span, "the IR call is malformed")
		return "nil"
	}
	x := generator.plotNumericListValue(value.Arguments[0].Value, meta)
	y := generator.plotNumericListValue(value.Arguments[1].Value, meta)
	z := generator.numericMatrixOf(value.Arguments[2].Value)
	return generator.plotSurfaceFrom("AhdPlotSurface("+plotErrorRuntime+", "+x+", "+y+", "+z+")", meta)
}

// plotSurfaceFrom wraps a runtime AhdSurface into a Surface value.
func (generator *generator) plotSurfaceFrom(surface string, meta ir.ExprBase) string {
	if generator.layouts[plotSurfaceClass] == nil {
		return generator.unsupported("a Surface value without its Class declaration", meta.Span)
	}
	name, known := generator.timeHelpers[plotSurfaceClass]
	if !known {
		name = mangleNamed("ps_", generator.classDisplayName(plotSurfaceClass), string(plotSurfaceClass))
		generator.timeHelpers[plotSurfaceClass] = name
	}
	return name + "(" + surface + ")"
}

// emitPlotSurfaceHelper writes the Surface wrapper once, when used.
func (generator *generator) emitPlotSurfaceHelper(writer *emitter) {
	name, known := generator.timeHelpers[plotSurfaceClass]
	layout := generator.layouts[plotSurfaceClass]
	if !known || layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	arguments := make([]string, len(plotSurfaceFields))
	for index, field := range plotSurfaceFields {
		arguments[index] = "surface." + field.runtime
	}
	writer.line("// Surface value built from one runtime surface reading.")
	writer.open("func " + name + "(surface AhdSurface) " + generator.interfaceName(plotSurfaceClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(" + strings.Join(arguments, ", ") + ")")
	writer.close("}")
	writer.blank()
}

// plotSurfaceOf evaluates one Surface expression once and reads its hidden
// fields into an AhdSurface.
func (generator *generator) plotSurfaceOf(expression ir.Expr) string {
	parts := make([]string, len(plotSurfaceFields))
	for index, field := range plotSurfaceFields {
		parts[index] = field.runtime + ": value." + generator.fieldName(plotSurfaceFieldID(field.field)) + "_get()"
	}
	return "func(value " + generator.interfaceName(plotSurfaceClass) + ") AhdSurface { return AhdSurface{" +
		strings.Join(parts, ", ") + "} }(" + generator.expr(expression) + ")"
}

// plotSurfaceOperation lowers the Surface members.
func (generator *generator) plotSurfaceOperation(name string, value *ir.CallExpr) string {
	generator.usesPlot = true
	meta := value.ExprMeta()
	if value.Callee == nil {
		generator.fail(CodeGenerationFailure, name+" has no receiver", meta.Span, "the IR call is malformed")
		return "nil"
	}
	argument := func(index int, kind ir.TypeKind, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: kind}, false)
	}
	surface := generator.plotSurfaceOf(value.Callee)
	switch name {
	case "Surface.title", "Surface.xLabel", "Surface.yLabel", "Surface.zLabel":
		setter := "AhdPlotSurface" + strings.ToUpper(name[len("Surface."):len("Surface.")+1]) + name[len("Surface.")+1:]
		return generator.plotSurfaceFrom(setter+"("+surface+", "+argument(0, ir.StringType, `""`)+")", meta)
	case "Surface.size":
		return generator.plotSurfaceFrom("AhdPlotSurfaceSize("+plotErrorRuntime+", "+surface+", "+
			argument(0, ir.IntType, "int64(0)")+", "+argument(1, ir.IntType, "int64(0)")+")", meta)
	case "Surface.wireframe":
		return generator.plotSurfaceFrom("AhdPlotSurfaceWireframe("+surface+", "+argument(0, ir.BoolType, "false")+")", meta)
	case "Surface.save":
		return "AhdPlotSurfaceSave(" + plotErrorRuntime + ", " + surface + ", " + argument(0, ir.StringType, `""`) + ")"
	case "Surface.show":
		return "AhdPlotSurfaceShow(" + plotErrorRuntime + ", " + surface + ")"
	}
	return generator.unsupported("Plot operation "+name, meta.Span)
}
