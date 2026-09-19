package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const plotModuleID = "builtin:Plot"

var (
	plotErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	plotErrorClass  = &types.ClassSymbol{ModuleID: plotModuleID, Name: "PlotError", Parent: plotErrorParent}
	plotChartClass  = &types.ClassSymbol{ModuleID: plotModuleID, Name: "Chart"}
	plotFigureClass = &types.ClassSymbol{ModuleID: plotModuleID, Name: "Figure"}
	// v2.0
	plotSurfaceClass = &types.ClassSymbol{ModuleID: plotModuleID, Name: "Surface"}
)

// PlotSurfaceIdentity exposes the Surface identity to the lowering layer.
func PlotSurfaceIdentity() *types.ClassSymbol { return plotSurfaceClass }

// PlotErrorIdentity, PlotChartIdentity, and PlotFigureIdentity expose the
// canonical identities to the lowering layer without coupling the public
// module interface to a backend.
func PlotErrorIdentity() *types.ClassSymbol  { return plotErrorClass }
func PlotChartIdentity() *types.ClassSymbol  { return plotChartClass }
func PlotFigureIdentity() *types.ClassSymbol { return plotFigureClass }

// PlotChartOperations and PlotFigureOperations name the members each Class
// publishes through built-in type operations, so has/has not reports what a
// Chart or Figure value really offers.
var (
	PlotChartOperations  = []string{"title", "xLabel", "yLabel", "legend", "size", "line", "scatter", "save", "show"}
	PlotFigureOperations = []string{"save", "show"}
	// PlotSurfaceOperations are a Surface's members (v2.0).
	PlotSurfaceOperations = []string{"title", "xLabel", "yLabel", "zLabel", "size", "wireframe", "show", "save"}
)

func plotChartType() types.Type  { return types.Class{Symbol: plotChartClass} }
func plotFigureType() types.Type { return types.Class{Symbol: plotFigureClass} }

// plotNumericElements are the two element types every Plot function accepts
// for a numeric List argument. Safe Int -> Real widening happens internally
// at the runtime layer; the frontend keeps both as distinct, exact overloads
// so the static element type is always known, matching the Statistics module
// convention.
var plotNumericElements = []types.Type{types.Int, types.Real}

// plotNumericSignatures builds every Int/Real combination for the named
// List<Int|Real> parameters, holding fixed any parameter named in `fixed`.
// This is how Plot.line/scatter/errorBar publish independently flexible
// numeric List arguments (e.g. List<Int> x with List<Real> y) without
// hand-writing 2^n signatures.
func plotNumericSignatures(result types.Type, names []string, fixed ...types.Parameter) []*types.Signature {
	combinations := plotNumericCombinations(len(names))
	signatures := make([]*types.Signature, 0, len(combinations))
	for _, combination := range combinations {
		parameters := make([]types.Parameter, 0, len(names)+len(fixed))
		for index, name := range names {
			parameters = append(parameters, types.Parameter{Name: name, Type: types.List{Element: combination[index]}})
		}
		parameters = append(parameters, fixed...)
		signatures = append(signatures, &types.Signature{Parameters: parameters, Return: result})
	}
	return signatures
}

func plotNumericCombinations(count int) [][]types.Type {
	if count == 0 {
		return [][]types.Type{{}}
	}
	rest := plotNumericCombinations(count - 1)
	combinations := make([][]types.Type, 0, len(rest)*len(plotNumericElements))
	for _, element := range plotNumericElements {
		for _, tail := range rest {
			combination := append([]types.Type{element}, tail...)
			combinations = append(combinations, combination)
		}
	}
	return combinations
}

// plotFunction publishes one Plot entry point with one or more signatures, an
// ordinary overload set resolved by the existing machinery, mirroring
// statisticsFunction.
func plotFunction(name string, signatures ...*types.Signature) *Symbol {
	symbol := &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: plotModuleID,
	}
	for _, signature := range signatures {
		callable := &Callable{
			Signature: signature, ParameterNull: nonNullParameters(len(signature.Parameters)),
			ReturnNull: NonNull,
		}
		if symbol.Callable == nil {
			symbol.Callable = callable
		}
		if len(signatures) > 1 {
			if symbol.OverloadSet == nil {
				symbol.OverloadSet = &OverloadSet{Name: name}
			}
			symbol.OverloadSet.Candidates = append(symbol.OverloadSet.Candidates, callable)
		}
	}
	if len(signatures) == 1 {
		symbol.Type = types.Function{Signature: signatures[0]}
	}
	return symbol
}

func plotModuleInterface() *ModuleInterface {
	module := standardInterface(plotModuleID, "Plot")

	errorSymbol := &Symbol{
		Name: "PlotError", Kind: ClassSymbol, Class: plotErrorClass,
		Type: types.Class{Symbol: plotErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: plotModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[plotModuleID+"\x00PlotError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	chartSymbol := &Symbol{
		Name: "Chart", Kind: ClassSymbol, Class: plotChartClass,
		Type: types.Class{Symbol: plotChartClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: plotModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[plotModuleID+"\x00Chart"] = chartSymbol
	addStandardExport(module, chartSymbol)

	figureSymbol := &Symbol{
		Name: "Figure", Kind: ClassSymbol, Class: plotFigureClass,
		Type: types.Class{Symbol: plotFigureClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: plotModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[plotModuleID+"\x00Figure"] = figureSymbol
	addStandardExport(module, figureSymbol)

	surfaceSymbol := &Symbol{
		Name: "Surface", Kind: ClassSymbol, Class: plotSurfaceClass,
		Type: types.Class{Symbol: plotSurfaceClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: plotModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[plotModuleID+"\x00Surface"] = surfaceSymbol
	addStandardExport(module, surfaceSymbol)
	// Plot.surface(x, y, z): x and y each List<Int>, List<Real>, or a
	// Vector; z a Matrix of one row per y and one column per x.
	var surfaceSignatures []*types.Signature
	for _, x := range plotSurfaceAxes() {
		for _, y := range plotSurfaceAxes() {
			surfaceSignatures = append(surfaceSignatures, &types.Signature{
				Parameters: []types.Parameter{{Name: "x", Type: x}, {Name: "y", Type: y}, {Name: "z", Type: numericMatrixType()}},
				Return:     types.Class{Symbol: plotSurfaceClass},
			})
		}
	}
	addStandardExport(module, plotFunction("surface", surfaceSignatures...))

	chart := plotChartType()
	addStandardExport(module, plotFunction("new", &types.Signature{Return: chart}))
	addStandardExport(module, plotFunction("line", plotNumericSignatures(chart, []string{"x", "y"})...))
	addStandardExport(module, plotFunction("scatter", plotNumericSignatures(chart, []string{"x", "y"})...))
	for _, name := range []string{"line", "scatter"} {
		symbol := module.Exports[name]
		callable := &Callable{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "x", Type: numericVectorType()}, {Name: "y", Type: numericVectorType()}}, Return: chart}, ParameterNull: []NullState{NonNull, NonNull}, ReturnNull: NonNull}
		symbol.OverloadSet.Candidates = append(symbol.OverloadSet.Candidates, callable)
	}
	addStandardExport(module, plotBarFunction())
	addStandardExport(module, plotFunction("histogram", plotNumericSignatures(chart, []string{"values"},
		types.Parameter{Name: "bins", Type: types.Int})...))
	addStandardExport(module, plotFunction("box", plotNumericSignatures(chart, []string{"values"})...))
	addStandardExport(module, plotFunction("errorBar",
		plotNumericSignatures(chart, []string{"x", "y", "lowerErrors", "upperErrors"})...))
	addStandardExport(module, plotFunction("subplots", &types.Signature{
		Parameters: []types.Parameter{
			{Name: "rows", Type: types.Int}, {Name: "columns", Type: types.Int},
			{Name: "charts", Type: types.List{Element: chart}},
		},
		Return: plotFigureType(),
	}))

	sort.Strings(module.ExportNames)
	return module
}

// plotBarFunction publishes Plot.bar(labels: List<String>, values:
// List<Int|Real>) -> Chart. It is built directly, rather than through
// plotNumericSignatures' fixed-parameters-last convention, so labels comes
// first in the publishable signature, matching the documented call shape.
func plotBarFunction() *Symbol {
	var signatures []*types.Signature
	for _, element := range plotNumericElements {
		signatures = append(signatures, &types.Signature{
			Parameters: []types.Parameter{
				{Name: "labels", Type: types.List{Element: types.String}},
				{Name: "values", Type: types.List{Element: element}},
			},
			Return: plotChartType(),
		})
	}
	return plotFunction("bar", signatures...)
}

// plotConstructionHint names the Plot functions and Chart members that
// produce a Chart or Figure, so direct construction has an actionable
// message instead of a generic missing-constructor diagnostic.
func plotConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != plotModuleID {
		return "", false
	}
	switch identity.Name {
	case "Chart":
		return "create a Chart with Plot.new, Plot.line, Plot.scatter, Plot.bar, Plot.histogram, " +
			"Plot.box, or Plot.errorBar, or derive one from an existing Chart", true
	case "Figure":
		return "create a Figure with Plot.subplots", true
	case "Surface":
		return "create a Surface with Plot.surface(x, y, z)", true
	default:
		return "", false
	}
}

// plotOperationFor names the built-in member a Chart or Figure instance
// publishes. Only the compiler-supplied identities match, so a user Class
// named Chart or Figure (impossible: standard modules cannot be shadowed)
// never collides with them.
func plotOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != plotModuleID {
		return "", false
	}
	switch class.Symbol.Name {
	case "Chart":
		operation, known := plotChartOperationNames[name]
		return operation, known
	case "Figure":
		operation, known := plotFigureOperationNames[name]
		return operation, known
	case "Surface":
		operation := TypeOperation("Surface." + name)
		_, known := plotMembers[operation]
		return operation, known
	default:
		return "", false
	}
}

var plotChartOperationNames = map[string]TypeOperation{
	"title": PlotChartTitle, "xLabel": PlotChartXLabel, "yLabel": PlotChartYLabel,
	"legend": PlotChartLegend, "size": PlotChartSize, "line": PlotChartLine,
	"scatter": PlotChartScatter, "save": PlotChartSave, "show": PlotChartShow,
}

var plotFigureOperationNames = map[string]TypeOperation{
	"save": PlotFigureSave, "show": PlotFigureShow,
}

// plotMembers publishes every Chart and Figure member as an ordinary Symbol
// with real parameter names, the way Canvas, Turtle, and GUI members do: the
// compiler checks a call against it, lowering binds its arguments by name,
// and the editor completes, hovers, and shows its signature from the same
// metadata. Chart.line and Chart.scatter accept List<Int>, List<Real>, or a
// Vector for x and y independently, as an ordinary overload set.
var plotMembers = func() map[TypeOperation]*Symbol {
	chart := plotChartType()
	surface := types.Type(types.Class{Symbol: plotSurfaceClass})
	p := func(name string, typ types.Type) types.Parameter { return types.Parameter{Name: name, Type: typ} }
	numeric := []types.Type{types.List{Element: types.Int}, types.List{Element: types.Real}, numericVectorType()}
	series := func(name string) *Symbol {
		var signatures []*types.Signature
		for _, x := range numeric {
			for _, y := range numeric {
				signatures = append(signatures, &types.Signature{
					Parameters: []types.Parameter{p("x", x), p("y", y), p("label", types.String)}, Return: chart})
			}
		}
		return completionOverloads(plotModuleID, name, signatures...)
	}
	return map[TypeOperation]*Symbol{
		PlotChartTitle:   completionMember(plotModuleID, "title", chart, p("text", types.String)),
		PlotChartXLabel:  completionMember(plotModuleID, "xLabel", chart, p("text", types.String)),
		PlotChartYLabel:  completionMember(plotModuleID, "yLabel", chart, p("text", types.String)),
		PlotChartLegend:  completionMember(plotModuleID, "legend", chart, p("enabled", types.Bool)),
		PlotChartSize:    completionMember(plotModuleID, "size", chart, p("width", types.Int), p("height", types.Int)),
		PlotChartLine:    series("line"),
		PlotChartScatter: series("scatter"),
		PlotChartSave:    completionMember(plotModuleID, "save", types.Nothing, p("path", types.String)),
		PlotChartShow:    completionMember(plotModuleID, "show", types.Nothing),
		PlotFigureSave:   completionMember(plotModuleID, "save", types.Nothing, p("path", types.String)),
		PlotFigureShow:   completionMember(plotModuleID, "show", types.Nothing),

		"Surface.title":     completionMember(plotModuleID, "title", surface, p("text", types.String)),
		"Surface.xLabel":    completionMember(plotModuleID, "xLabel", surface, p("text", types.String)),
		"Surface.yLabel":    completionMember(plotModuleID, "yLabel", surface, p("text", types.String)),
		"Surface.zLabel":    completionMember(plotModuleID, "zLabel", surface, p("text", types.String)),
		"Surface.size":      completionMember(plotModuleID, "size", surface, p("width", types.Int), p("height", types.Int)),
		"Surface.wireframe": completionMember(plotModuleID, "wireframe", surface, p("enabled", types.Bool)),
		"Surface.show":      completionMember(plotModuleID, "show", types.Nothing),
		"Surface.save":      completionMember(plotModuleID, "save", types.Nothing, p("path", types.String)),
	}
}()

// plotSurfaceAxes are the types Plot.surface accepts for x and y.
func plotSurfaceAxes() []types.Type {
	return []types.Type{types.List{Element: types.Int}, types.List{Element: types.Real}, numericVectorType()}
}
