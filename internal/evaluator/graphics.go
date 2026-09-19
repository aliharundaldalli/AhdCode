package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The Graphics standard module's REPL implementation. It calls the same
// runtime functions a compiled program calls, so validation, color parsing,
// Turtle geometry, and the helper protocol are one implementation. A Canvas
// or Turtle value is an Instance holding its runtime handle.

const (
	graphicsCanvasClassID = ir.ClassID("builtin:Graphics::class::Canvas")
	graphicsTurtleClassID = ir.ClassID("builtin:Graphics::class::Turtle")
	graphicsCanvasHandle  = ir.FieldID("builtin:Graphics::class::Canvas::field::handle")
	graphicsTurtleHandle  = ir.FieldID("builtin:Graphics::class::Turtle::field::handle")
)

func (session *Session) graphicsCheck(problem string) {
	if problem != "" {
		session.raise("GraphicsError", problem)
	}
}

// graphicsHandler reads the Function argument of an event registration.
func (session *Session) graphicsHandler(arguments []any, member string) *FunctionValue {
	if len(arguments) > 0 {
		if function, ok := arguments[0].(*FunctionValue); ok && function != nil {
			return function
		}
	}
	session.raise("NullError", member+" needs a Function")
	return nil
}

// invokeHandler runs one event callback, the same way a compiled program
// calls its Go func: its result is discarded, its error propagates, and what
// it writes is flushed at once.
func (session *Session) invokeHandler(handler *FunctionValue, values ...any) {
	arguments := make([]argumentValue, len(values))
	for index, value := range values {
		arguments[index] = argumentValue{value: value}
	}
	session.invoke(handler, arguments)
	if flusher, ok := session.Output.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
}

func graphicsInstance(class ir.ClassID, field ir.FieldID, handle int64) *Instance {
	return &Instance{Class: class, Fields: map[ir.FieldID]any{field: handle}}
}

func (session *Session) graphicsHandle(receiver any, field ir.FieldID) int64 {
	handle, _ := session.requireInstance(receiver).Fields[field].(int64)
	return handle
}

// graphicsBuiltin implements Graphics.open. Omitted defaults arrive as nil.
func (session *Session) graphicsBuiltin(name string, arguments []any) any {
	if name != "open" {
		session.raise("Error", "unsupported Graphics function "+name)
	}
	width, height := graphicsInt(arguments, 0, 800), graphicsInt(arguments, 1, 600)
	title, background := graphicsText(arguments, 2, "AhdCode Graphics"), graphicsText(arguments, 3, "white")
	handle, problem := ahdruntime.AhdGraphicsOpen(width, height, title, background)
	session.graphicsCheck(problem)
	return graphicsInstance(graphicsCanvasClassID, graphicsCanvasHandle, handle)
}

// graphicsOperation implements the members of Canvas and Turtle.
func (session *Session) graphicsOperation(name string, receiver any, arguments []any) any {
	real := func(index int, fallback float64) float64 { return graphicsReal(arguments, index, fallback) }
	text := func(index int, fallback string) string { return graphicsText(arguments, index, fallback) }
	nullable := func(index int) *string {
		if index < len(arguments) {
			if value, ok := arguments[index].(string); ok {
				return &value
			}
		}
		return nil
	}
	if len(name) > len("Canvas.") && name[:len("Canvas.")] == "Canvas." {
		canvas := session.graphicsHandle(receiver, graphicsCanvasHandle)
		switch name {
		case "Canvas.clear":
			session.graphicsCheck(ahdruntime.AhdGraphicsClear(canvas, text(0, "white")))
		case "Canvas.line":
			session.graphicsCheck(ahdruntime.AhdGraphicsLine(canvas, real(0, 0), real(1, 0), real(2, 0), real(3, 0), text(4, "black"), real(5, 1)))
		case "Canvas.circle":
			session.graphicsCheck(ahdruntime.AhdGraphicsCircle(canvas, real(0, 0), real(1, 0), real(2, 0), text(3, "black"), nullable(4), real(5, 1)))
		case "Canvas.rectangle":
			session.graphicsCheck(ahdruntime.AhdGraphicsRectangle(canvas, real(0, 0), real(1, 0), real(2, 0), real(3, 0), text(4, "black"), nullable(5), real(6, 1)))
		case "Canvas.save":
			session.graphicsCheck(ahdruntime.AhdGraphicsSave(canvas, text(0, "")))
		case "Canvas.onClick":
			handler := session.graphicsHandler(arguments, "Canvas.onClick")
			session.graphicsCheck(ahdruntime.AhdGraphicsOnClick(canvas, func(x, y float64) {
				session.invokeHandler(handler, x, y)
			}))
		case "Canvas.onKey":
			handler := session.graphicsHandler(arguments, "Canvas.onKey")
			session.graphicsCheck(ahdruntime.AhdGraphicsOnKey(canvas, func(key string) {
				session.invokeHandler(handler, key)
			}))
		case "Canvas.wait":
			// Output written before wait must be visible while the window is open.
			session.flushOutput()
			session.graphicsCheck(ahdruntime.AhdGraphicsWait(canvas))
		case "Canvas.close":
			session.graphicsCheck(ahdruntime.AhdGraphicsClose(canvas))
		case "Canvas.isOpen":
			return ahdruntime.AhdGraphicsIsOpen(canvas)
		case "Canvas.turtle":
			handle, problem := ahdruntime.AhdGraphicsTurtle(canvas)
			session.graphicsCheck(problem)
			return graphicsInstance(graphicsTurtleClassID, graphicsTurtleHandle, handle)
		default:
			session.raise("Error", "unsupported Graphics operation "+name)
		}
		return Nothing
	}
	turtle := session.graphicsHandle(receiver, graphicsTurtleHandle)
	switch name {
	case "Turtle.forward":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleForward(turtle, real(0, 0)))
	case "Turtle.backward":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleBackward(turtle, real(0, 0)))
	case "Turtle.left":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleLeft(turtle, real(0, 0)))
	case "Turtle.right":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleRight(turtle, real(0, 0)))
	case "Turtle.moveTo":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleMoveTo(turtle, real(0, 0), real(1, 0)))
	case "Turtle.setHeading":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleSetHeading(turtle, real(0, 0)))
	case "Turtle.penUp":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtlePen(turtle, false))
	case "Turtle.penDown":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtlePen(turtle, true))
	case "Turtle.setColor":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleSetColor(turtle, text(0, "")))
	case "Turtle.setWidth":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleSetWidth(turtle, real(0, 0)))
	case "Turtle.home":
		session.graphicsCheck(ahdruntime.AhdGraphicsTurtleHome(turtle))
	case "Turtle.x", "Turtle.y", "Turtle.heading":
		x, y, heading, problem := ahdruntime.AhdGraphicsTurtleState(turtle)
		session.graphicsCheck(problem)
		switch name {
		case "Turtle.x":
			return x
		case "Turtle.y":
			return y
		}
		return heading
	default:
		session.raise("Error", "unsupported Graphics operation "+name)
	}
	return Nothing
}

// CloseGraphics closes every Canvas this process opened. The REPL calls it
// when a session ends so no window outlives it.
func CloseGraphics() { ahdruntime.AhdGraphicsCloseAll() }

func graphicsReal(arguments []any, index int, fallback float64) float64 {
	if index < len(arguments) {
		switch value := arguments[index].(type) {
		case float64:
			return value
		case int64:
			return float64(value)
		}
	}
	return fallback
}

func graphicsInt(arguments []any, index int, fallback int64) int64 {
	if index < len(arguments) {
		if value, ok := arguments[index].(int64); ok {
			return value
		}
	}
	return fallback
}

func graphicsText(arguments []any, index int, fallback string) string {
	if index < len(arguments) {
		if value, ok := arguments[index].(string); ok {
			return value
		}
	}
	return fallback
}
