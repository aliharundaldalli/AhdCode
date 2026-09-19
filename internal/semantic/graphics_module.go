package semantic

import (
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const graphicsModuleID = "builtin:Graphics"

var (
	graphicsErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	graphicsErrorClass  = &types.ClassSymbol{ModuleID: graphicsModuleID, Name: "GraphicsError", Parent: graphicsErrorParent}
	graphicsCanvasClass = &types.ClassSymbol{ModuleID: graphicsModuleID, Name: "Canvas"}
	graphicsTurtleClass = &types.ClassSymbol{ModuleID: graphicsModuleID, Name: "Turtle"}
)

// GraphicsErrorIdentity, GraphicsCanvasIdentity, and GraphicsTurtleIdentity
// expose the canonical identities to the lowering layer.
func GraphicsErrorIdentity() *types.ClassSymbol  { return graphicsErrorClass }
func GraphicsCanvasIdentity() *types.ClassSymbol { return graphicsCanvasClass }
func GraphicsTurtleIdentity() *types.ClassSymbol { return graphicsTurtleClass }

func graphicsCanvasType() types.Type { return types.Class{Symbol: graphicsCanvasClass} }
func graphicsTurtleType() types.Type { return types.Class{Symbol: graphicsTurtleClass} }

// ClickHandlerType is the (x: Real, y: Real) -> Nothing shape of a Canvas
// click callback; KeyHandlerType is the (key: String) -> Nothing shape of a
// Canvas or Window key callback.
func ClickHandlerType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "x", Type: types.Real}, {Name: "y", Type: types.Real}}, Return: types.Nothing}}
}

func KeyHandlerType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "key", Type: types.String}}, Return: types.Nothing}}
}

// The built-in members of Canvas and Turtle. Unlike the fixed-shape members
// of other built-in Classes, these publish real parameter names and defaults,
// so a call may be entirely positional or entirely named, exactly like a call
// to a module function such as Graphics.open.
const (
	GraphicsCanvasClear     TypeOperation = "Canvas.clear"
	GraphicsCanvasLine      TypeOperation = "Canvas.line"
	GraphicsCanvasCircle    TypeOperation = "Canvas.circle"
	GraphicsCanvasRectangle TypeOperation = "Canvas.rectangle"
	GraphicsCanvasSave      TypeOperation = "Canvas.save"
	GraphicsCanvasWait      TypeOperation = "Canvas.wait"
	GraphicsCanvasClose     TypeOperation = "Canvas.close"
	GraphicsCanvasIsOpen    TypeOperation = "Canvas.isOpen"
	GraphicsCanvasTurtle    TypeOperation = "Canvas.turtle"
	GraphicsCanvasOnClick   TypeOperation = "Canvas.onClick"
	GraphicsCanvasOnKey     TypeOperation = "Canvas.onKey"

	GraphicsTurtleForward    TypeOperation = "Turtle.forward"
	GraphicsTurtleBackward   TypeOperation = "Turtle.backward"
	GraphicsTurtleLeft       TypeOperation = "Turtle.left"
	GraphicsTurtleRight      TypeOperation = "Turtle.right"
	GraphicsTurtleMoveTo     TypeOperation = "Turtle.moveTo"
	GraphicsTurtleSetHeading TypeOperation = "Turtle.setHeading"
	GraphicsTurtlePenUp      TypeOperation = "Turtle.penUp"
	GraphicsTurtlePenDown    TypeOperation = "Turtle.penDown"
	GraphicsTurtleSetColor   TypeOperation = "Turtle.setColor"
	GraphicsTurtleSetWidth   TypeOperation = "Turtle.setWidth"
	GraphicsTurtleHome       TypeOperation = "Turtle.home"
	GraphicsTurtleX          TypeOperation = "Turtle.x"
	GraphicsTurtleY          TypeOperation = "Turtle.y"
	GraphicsTurtleHeading    TypeOperation = "Turtle.heading"
)

// GraphicsCanvasOperations and GraphicsTurtleOperations name the members
// each Class publishes, so has/has not reports what a value really offers.
var (
	GraphicsCanvasOperations = []string{"clear", "line", "circle", "rectangle", "save", "wait", "close", "isOpen", "turtle", "onClick", "onKey"}
	GraphicsTurtleOperations = []string{"forward", "backward", "left", "right", "moveTo", "setHeading",
		"penUp", "penDown", "setColor", "setWidth", "home", "x", "y", "heading"}
)

// graphicsParameter describes one member parameter. nullable marks the one
// parameter that accepts null (a shape's fill).
type graphicsParameter struct {
	name     string
	typ      types.Type
	optional bool
	nullable bool
}

func graphicsRequired(name string, typ types.Type) graphicsParameter {
	return graphicsParameter{name: name, typ: typ}
}

func graphicsOptional(name string, typ types.Type) graphicsParameter {
	return graphicsParameter{name: name, typ: typ, optional: true}
}

// graphicsMember builds the Symbol that describes one built-in member: its
// Callable is what the call is checked against, what signature help shows,
// and what hover renders.
func graphicsMember(name string, result types.Type, parameters ...graphicsParameter) *Symbol {
	signature := &types.Signature{Return: result}
	nulls := make([]NullState, len(parameters))
	for index, parameter := range parameters {
		signature.Parameters = append(signature.Parameters,
			types.Parameter{Name: parameter.name, Type: parameter.typ, HasDefault: parameter.optional})
		nulls[index] = NonNull
		if parameter.nullable {
			nulls[index] = MaybeNull
		}
	}
	return &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		Builtin: true, InitialNull: NonNull, OriginModuleID: graphicsModuleID,
		Callable: &Callable{Signature: signature, ParameterNull: nulls, ReturnNull: NonNull},
	}
}

var graphicsMembers = func() map[TypeOperation]*Symbol {
	real := func(name string) graphicsParameter { return graphicsRequired(name, types.Real) }
	fill := graphicsParameter{name: "fill", typ: types.String, optional: true, nullable: true}
	return map[TypeOperation]*Symbol{
		GraphicsCanvasClear: graphicsMember("clear", types.Nothing, graphicsOptional("color", types.String)),
		GraphicsCanvasLine: graphicsMember("line", types.Nothing, real("x1"), real("y1"), real("x2"), real("y2"),
			graphicsOptional("color", types.String), graphicsOptional("width", types.Real)),
		GraphicsCanvasCircle: graphicsMember("circle", types.Nothing, real("x"), real("y"), real("radius"),
			graphicsOptional("stroke", types.String), fill, graphicsOptional("width", types.Real)),
		GraphicsCanvasRectangle: graphicsMember("rectangle", types.Nothing, real("x"), real("y"), real("width"), real("height"),
			graphicsOptional("stroke", types.String), fill, graphicsOptional("lineWidth", types.Real)),
		GraphicsCanvasSave:   graphicsMember("save", types.Nothing, graphicsRequired("path", types.String)),
		GraphicsCanvasWait:   graphicsMember("wait", types.Nothing),
		GraphicsCanvasClose:  graphicsMember("close", types.Nothing),
		GraphicsCanvasIsOpen: graphicsMember("isOpen", types.Bool),
		GraphicsCanvasTurtle: graphicsMember("turtle", graphicsTurtleType()),
		// v1.8: one callback per event kind, run by Canvas.wait. The handler
		// shapes are checked statically like any Function argument.
		GraphicsCanvasOnClick: graphicsMember("onClick", types.Nothing, graphicsRequired("handler", ClickHandlerType())),
		GraphicsCanvasOnKey:   graphicsMember("onKey", types.Nothing, graphicsRequired("handler", KeyHandlerType())),

		GraphicsTurtleForward:    graphicsMember("forward", types.Nothing, real("distance")),
		GraphicsTurtleBackward:   graphicsMember("backward", types.Nothing, real("distance")),
		GraphicsTurtleLeft:       graphicsMember("left", types.Nothing, real("degrees")),
		GraphicsTurtleRight:      graphicsMember("right", types.Nothing, real("degrees")),
		GraphicsTurtleMoveTo:     graphicsMember("moveTo", types.Nothing, real("x"), real("y")),
		GraphicsTurtleSetHeading: graphicsMember("setHeading", types.Nothing, real("degrees")),
		GraphicsTurtlePenUp:      graphicsMember("penUp", types.Nothing),
		GraphicsTurtlePenDown:    graphicsMember("penDown", types.Nothing),
		GraphicsTurtleSetColor:   graphicsMember("setColor", types.Nothing, graphicsRequired("color", types.String)),
		GraphicsTurtleSetWidth:   graphicsMember("setWidth", types.Nothing, real("width")),
		GraphicsTurtleHome:       graphicsMember("home", types.Nothing),
		GraphicsTurtleX:          graphicsMember("x", types.Real),
		GraphicsTurtleY:          graphicsMember("y", types.Real),
		GraphicsTurtleHeading:    graphicsMember("heading", types.Real),
	}
}()

func graphicsModuleInterface() *ModuleInterface {
	module := standardInterface(graphicsModuleID, "Graphics")

	errorSymbol := &Symbol{
		Name: "GraphicsError", Kind: ClassSymbol, Class: graphicsErrorClass,
		Type: types.Class{Symbol: graphicsErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: graphicsModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[graphicsModuleID+"\x00GraphicsError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	for _, class := range []*types.ClassSymbol{graphicsCanvasClass, graphicsTurtleClass} {
		symbol := &Symbol{
			Name: class.Name, Kind: ClassSymbol, Class: class,
			Type: types.Class{Symbol: class, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: graphicsModuleID,
			Members: make(map[string]*Symbol),
		}
		module.Classes[graphicsModuleID+"\x00"+class.Name] = symbol
		addStandardExport(module, symbol)
	}

	addStandardExport(module, standardFunction(graphicsModuleID, "open", graphicsCanvasType(),
		types.Parameter{Name: "width", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "height", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "title", Type: types.String, HasDefault: true},
		types.Parameter{Name: "background", Type: types.String, HasDefault: true}))

	sort.Strings(module.ExportNames)
	return module
}

// graphicsOperationFor names the built-in member a Canvas or Turtle value
// publishes. Only the compiler-supplied identities match.
func graphicsOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != graphicsModuleID {
		return "", false
	}
	operation := TypeOperation(class.Symbol.Name + "." + name)
	_, known := graphicsMembers[operation]
	return operation, known
}

// TypeOperationBindsArguments reports whether a built-in member publishes
// parameter names and defaults, so lowering orders its arguments by name and
// marks omitted defaults, as it does for a module function call.
func TypeOperationBindsArguments(operation TypeOperation) bool {
	return publishedMember(operation) != nil
}

// BuiltinClassMembers lists the built-in members a compiler-supplied Class
// publishes as ordinary Symbols with real signatures, for editor completion.
// Classes whose members have no published signature report none.
func BuiltinClassMembers(identity *types.ClassSymbol) []*Symbol {
	if identity == nil {
		return nil
	}
	names := publishedMemberNames(identity)
	members := make([]*Symbol, 0, len(names))
	for _, name := range names {
		members = append(members, publishedMember(TypeOperation(identity.Name+"."+name)))
	}
	return members
}

// graphicsConstructionHint names how a Canvas or Turtle is obtained, since
// neither Class publishes a constructor.
func graphicsConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != graphicsModuleID {
		return "", false
	}
	switch identity.Name {
	case "Canvas":
		return "open a Canvas with Graphics.open", true
	case "Turtle":
		return "get a Turtle from an open Canvas with canvas.turtle()", true
	}
	return "", false
}

// analyzeGraphicsOperation checks one Canvas or Turtle member call against
// its published Callable with the same argument binder an ordinary call
// uses, and records that Callable so lowering and signature help see the
// exact parameter list. Hover on the member resolves to its Symbol.
func (a *analyzer) analyzeGraphicsOperation(call *ast.CallExpr, member *ast.MemberExpr, operation TypeOperation, current *scope, flow flowState) expressionInfo {
	symbol := publishedMember(operation)
	a.result.ResolvedSymbols[member] = symbol
	if symbol.OverloadSet != nil && len(symbol.OverloadSet.Candidates) > 1 {
		// An overloaded member (Table.innerJoin) selects its signature with
		// the ordinary overload resolution a module function call uses.
		selected := a.resolveOverloadCall(call, symbol.OverloadSet, a.analyzeCallArguments(call, current, flow, nil))
		if selected == nil {
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
		}
		a.result.SelectedCallables[call] = selected
		return expressionInfo{typeValue: selected.Signature.Return, nullState: NonNull}
	}
	callable := symbol.Callable
	parameters := callable.Signature.Parameters
	named := len(call.Arguments) > 0 && call.Arguments[0].Name != ""
	arguments := make([]expressionInfo, len(call.Arguments))
	for index, argument := range call.Arguments {
		target := index
		if named {
			target = -1
			for candidate := range parameters {
				if parameters[candidate].Name == argument.Name {
					target = candidate
					break
				}
			}
		}
		if target >= 0 && target < len(parameters) {
			arguments[index] = a.analyzeExpressionExpected(argument.Value, current, flow, parameters[target].Type)
		} else {
			arguments[index] = a.analyzeExpression(argument.Value, current, flow)
		}
	}
	a.callArgumentsCompatible(call, callable, arguments, true)
	a.result.SelectedCallables[call] = callable
	a.result.ResolvedSymbols[member] = symbol
	return expressionInfo{typeValue: callable.Signature.Return, nullState: NonNull}
}
