package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const graphicsModulePrefix = "builtin:Graphics::"

const (
	graphicsCanvasClass = ir.ClassID("builtin:Graphics::class::Canvas")
	graphicsTurtleClass = ir.ClassID("builtin:Graphics::class::Turtle")
)

func graphicsHandleField(class ir.ClassID) ir.FieldID {
	return ir.FieldID(string(class) + "::field::handle")
}

// graphicsArguments renders one Graphics call's arguments. Lowering has
// already ordered them by parameter name; an omitted default arrives with no
// value and takes its documented default here.
type graphicsArguments struct {
	generator *generator
	value     *ir.CallExpr
}

func (arguments graphicsArguments) present(index int) bool {
	return index < len(arguments.value.Arguments) && arguments.value.Arguments[index].Value != nil
}

func (arguments graphicsArguments) real(index int, fallback string) string {
	if !arguments.present(index) {
		return fallback
	}
	return arguments.generator.value(arguments.value.Arguments[index].Value, ir.Type{Kind: ir.RealType}, false)
}

func (arguments graphicsArguments) integer(index int, fallback string) string {
	if !arguments.present(index) {
		return fallback
	}
	return arguments.generator.value(arguments.value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
}

func (arguments graphicsArguments) text(index int, fallback string) string {
	if !arguments.present(index) {
		return fallback
	}
	return arguments.generator.value(arguments.value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
}

// nullableText renders a String? argument as *string: nil when omitted or null.
func (arguments graphicsArguments) nullableText(index int) string {
	if !arguments.present(index) {
		return "nil"
	}
	return arguments.generator.value(arguments.value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, true)
}

// graphicsValue wraps a runtime handle into a Canvas or Turtle value through
// the Class's generated handle constructor.
func (generator *generator) graphicsValue(class ir.ClassID, handle string, meta ir.ExprBase) string {
	layout := generator.layouts[class]
	if layout == nil {
		return generator.unsupported("a Graphics value without its Class declaration", meta.Span)
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return generator.unsupported("a Graphics value without its Class declaration", meta.Span)
	}
	return generator.callableName(constructor) + "(" + handle + ")"
}

// graphicsCall lowers Graphics.open.
func (generator *generator) graphicsCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), graphicsModulePrefix)
	arguments := graphicsArguments{generator: generator, value: value}
	switch name {
	case "open":
		handle := "AhdGraphicsOpenChecked(" + arguments.integer(0, "int64(800)") + ", " + arguments.integer(1, "int64(600)") + ", " +
			arguments.text(2, `"AhdCode Graphics"`) + ", " + arguments.text(3, `"white"`) + ")"
		generator.usesGraphics = true
		return generator.graphicsValue(graphicsCanvasClass, handle, meta)
	default:
		return generator.unsupported("Graphics function "+name, meta.Span)
	}
}

// graphicsOperation lowers the members of Canvas and Turtle. The receiver is
// read once, for its handle; each member is one runtime call, and a reported
// problem becomes a GraphicsError through AhdGraphicsCheck.
func (generator *generator) graphicsOperation(name string, value *ir.CallExpr) string {
	generator.usesGraphics = true
	meta := value.ExprMeta()
	if value.Callee == nil {
		generator.fail(CodeGenerationFailure, name+" has no receiver", meta.Span, "the IR call is malformed")
		return "nil"
	}
	class := graphicsCanvasClass
	if strings.HasPrefix(name, "Turtle.") {
		class = graphicsTurtleClass
	}
	handle := generator.expr(value.Callee) + "." + generator.fieldName(graphicsHandleField(class)) + "_get()"
	arguments := graphicsArguments{generator: generator, value: value}
	check := func(call string) string { return "AhdGraphicsCheck(" + call + ")" }
	switch name {
	case "Canvas.clear":
		return check("AhdGraphicsClear(" + handle + ", " + arguments.text(0, `"white"`) + ")")
	case "Canvas.line":
		return check("AhdGraphicsLine(" + handle + ", " + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ", " +
			arguments.real(2, "0") + ", " + arguments.real(3, "0") + ", " + arguments.text(4, `"black"`) + ", " + arguments.real(5, "1.0") + ")")
	case "Canvas.circle":
		return check("AhdGraphicsCircle(" + handle + ", " + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ", " +
			arguments.real(2, "0") + ", " + arguments.text(3, `"black"`) + ", " + arguments.nullableText(4) + ", " + arguments.real(5, "1.0") + ")")
	case "Canvas.rectangle":
		return check("AhdGraphicsRectangle(" + handle + ", " + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ", " +
			arguments.real(2, "0") + ", " + arguments.real(3, "0") + ", " + arguments.text(4, `"black"`) + ", " +
			arguments.nullableText(5) + ", " + arguments.real(6, "1.0") + ")")
	case "Canvas.save":
		return check("AhdGraphicsSave(" + handle + ", " + arguments.text(0, `""`) + ")")
	case "Canvas.onClick", "Canvas.onKey":
		// The handler's static type is exactly (Real, Real) -> Nothing or
		// (String) -> Nothing, which the generated code represents as the Go
		// func type the runtime stores.
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		if name == "Canvas.onClick" {
			return "AhdGraphicsOnClickChecked(" + handle + ", " + generator.adaptHandler(value.Arguments[0].Value, ir.Type{Kind: ir.RealType}, ir.Type{Kind: ir.RealType}) + ")"
		}
		return "AhdGraphicsOnKeyChecked(" + handle + ", " + generator.adaptHandler(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}) + ")"
	case "Canvas.wait":
		return check("AhdGraphicsWait(" + handle + ")")
	case "Canvas.close":
		return check("AhdGraphicsClose(" + handle + ")")
	case "Canvas.isOpen":
		return "AhdGraphicsIsOpen(" + handle + ")"
	case "Canvas.turtle":
		return generator.graphicsValue(graphicsTurtleClass, "AhdGraphicsTurtleChecked("+handle+")", meta)

	case "Turtle.forward":
		return check("AhdGraphicsTurtleForward(" + handle + ", " + arguments.real(0, "0") + ")")
	case "Turtle.backward":
		return check("AhdGraphicsTurtleBackward(" + handle + ", " + arguments.real(0, "0") + ")")
	case "Turtle.left":
		return check("AhdGraphicsTurtleLeft(" + handle + ", " + arguments.real(0, "0") + ")")
	case "Turtle.right":
		return check("AhdGraphicsTurtleRight(" + handle + ", " + arguments.real(0, "0") + ")")
	case "Turtle.moveTo":
		return check("AhdGraphicsTurtleMoveTo(" + handle + ", " + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ")")
	case "Turtle.setHeading":
		return check("AhdGraphicsTurtleSetHeading(" + handle + ", " + arguments.real(0, "0") + ")")
	case "Turtle.penUp":
		return check("AhdGraphicsTurtlePen(" + handle + ", false)")
	case "Turtle.penDown":
		return check("AhdGraphicsTurtlePen(" + handle + ", true)")
	case "Turtle.setColor":
		return check("AhdGraphicsTurtleSetColor(" + handle + ", " + arguments.text(0, `""`) + ")")
	case "Turtle.setWidth":
		return check("AhdGraphicsTurtleSetWidth(" + handle + ", " + arguments.real(0, "0") + ")")
	case "Turtle.home":
		return check("AhdGraphicsTurtleHome(" + handle + ")")
	case "Turtle.x":
		return "AhdGraphicsTurtleXChecked(" + handle + ")"
	case "Turtle.y":
		return "AhdGraphicsTurtleYChecked(" + handle + ")"
	case "Turtle.heading":
		return "AhdGraphicsTurtleHeadingChecked(" + handle + ")"
	}
	return generator.unsupported("Graphics operation "+name, meta.Span)
}
