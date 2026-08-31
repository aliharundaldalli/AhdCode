package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const plotModuleID = "builtin:Plot"

var (
	plotErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	plotErrorClass  = &types.ClassSymbol{ModuleID: plotModuleID, Name: "PlotError", Parent: plotErrorParent}
	plotChartClass  = &types.ClassSymbol{ModuleID: plotModuleID, Name: "Chart"}
	plotFigureClass = &types.ClassSymbol{ModuleID: plotModuleID, Name: "Figure"}
)

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

	chart := plotChartType()
	addStandardExport(module, plotFunction("new", &types.Signature{Return: chart}))
	addStandardExport(module, plotFunction("line", plotNumericSignatures(chart, []string{"x", "y"})...))
	addStandardExport(module, plotFunction("scatter", plotNumericSignatures(chart, []string{"x", "y"})...))
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

// plotSeriesOperation reports whether an operation is Chart.line or
// Chart.scatter, which need the hand-written numeric-List-flexible check
// below rather than the fixed-shape table.
func plotSeriesOperation(operation TypeOperation) bool {
	return operation == PlotChartLine || operation == PlotChartScatter
}

// plotOperationShape is the fixed call shape of one Chart or Figure member
// whose arguments are plain, single-typed values.
type plotOperationShape struct {
	parameters []types.Type
	result     types.Type
	hint       string
}

func plotOperationShapes() map[TypeOperation]plotOperationShape {
	chart := plotChartType()
	return map[TypeOperation]plotOperationShape{
		PlotChartTitle:  {[]types.Type{types.String}, chart, "pass one String title"},
		PlotChartXLabel: {[]types.Type{types.String}, chart, "pass one String x-axis label"},
		PlotChartYLabel: {[]types.Type{types.String}, chart, "pass one String y-axis label"},
		PlotChartLegend: {[]types.Type{types.Bool}, chart, "pass true or false"},
		PlotChartSize:   {[]types.Type{types.Int, types.Int}, chart, "pass a positive Int width and height"},
		PlotChartSave:   {[]types.Type{types.String}, types.Nothing, "pass a destination path ending in .png, .svg, or .pdf"},
		PlotChartShow:   {[]types.Type{}, types.Nothing, "call show with no argument"},
		PlotFigureSave:  {[]types.Type{types.String}, types.Nothing, "pass a destination path ending in .png, .svg, or .pdf"},
		PlotFigureShow:  {[]types.Type{}, types.Nothing, "call show with no argument"},
	}
}

// analyzePlotOperation checks one fixed-shape Chart or Figure member.
func (a *analyzer) analyzePlotOperation(call *ast.CallExpr, operation TypeOperation, shape plotOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, expected := range shape.parameters {
		argument := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
		if argument.invalid() {
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[index].Value, argument.nullState)
			continue
		}
		if !types.Assignable(expected, argument.typeValue) {
			a.typeMismatch(call.Arguments[index].Span(), expected, argument.typeValue, string(operation)+" argument")
		}
	}
	return result
}

// analyzePlotSeriesOperation checks Chart.line(x, y, label) and
// Chart.scatter(x, y, label). x and y each independently accept List<Int> or
// List<Real>, matching the flexibility Plot.line/Plot.scatter get through
// ordinary overload resolution -- a fixed single-shape table cannot express
// that, so this mirrors Data's hand-written analyzeDataSort dual-shape check.
func (a *analyzer) analyzePlotSeriesOperation(call *ast.CallExpr, operation TypeOperation, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: plotChartType(), nullState: NonNull}
	if len(call.Arguments) != 3 {
		a.error(codeCallArguments, fmt.Sprintf("%s expects 3 argument(s); received %d", operation, len(call.Arguments)),
			call.Span(), "pass an x List, a y List, and a String label")
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	a.requirePlotNumericList(call, 0, operation, current, flow)
	a.requirePlotNumericList(call, 1, operation, current, flow)
	label := a.analyzeExpressionExpected(call.Arguments[2].Value, current, flow, types.String)
	if !label.invalid() {
		if label.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[2].Value, label.nullState)
		} else if !types.Assignable(types.String, label.typeValue) {
			a.typeMismatch(call.Arguments[2].Span(), types.String, label.typeValue, string(operation)+" label")
		}
	}
	return result
}

// requirePlotNumericList checks that one argument is a NonNull List<Int> or
// List<Real>. It underlies every Plot operation that accepts a numeric List
// through a TypeOperation rather than through ordinary overload resolution.
func (a *analyzer) requirePlotNumericList(call *ast.CallExpr, index int, operation TypeOperation, current *scope, flow flowState) {
	reported := a.bag.Len()
	info := a.analyzeExpression(call.Arguments[index].Value, current, flow)
	if info.invalid() || a.bag.Len() != reported {
		return
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[index].Value, info.nullState)
		return
	}
	list, isList := info.typeValue.(types.List)
	numeric := isList && list.Element != nil && (list.Element.Kind() == types.IntKind || list.Element.Kind() == types.RealKind)
	if !numeric {
		a.typeMismatch(call.Arguments[index].Span(), types.List{Element: types.Real}, info.typeValue, string(operation)+" argument")
	}
}
