package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const plotModulePrefix = "builtin:Plot::"

const (
	plotChartClass  = ir.ClassID("builtin:Plot::class::Chart")
	plotFigureClass = ir.ClassID("builtin:Plot::class::Figure")
)

// The runtime Class Plot's helpers raise through. It is named directly
// rather than through a generated descriptor for the same reason Statistics
// and Data are: Plot can be imported without the Class being separately
// declared.
const plotErrorRuntime = "AhdClassPlotError"

// plotFields lists the Chart Class's storage fields in the exact order
// internal/lowering/plot_builtin.go declares them -- the same order its
// all-fields constructor expects them positionally, and the order
// emitPlotHelpers below must supply them in.
var plotFields = []string{
	"kind",
	"seriesKinds", "seriesLabels", "seriesX", "seriesY",
	"barLabels", "barValues",
	"histogramValues", "histogramBins",
	"boxValues",
	"errorX", "errorY", "errorLower", "errorUpper",
	"title", "xLabel", "yLabel", "legend", "width", "height",
}

func plotFieldID(name string) ir.FieldID {
	return ir.FieldID(string(plotChartClass) + "::field::" + name)
}

func plotFigureFieldID(name string) ir.FieldID {
	return ir.FieldID(string(plotFigureClass) + "::field::" + name)
}

// plotCall lowers the Plot standard module's module-root functions. The
// frontend has already selected the Int/Real overload for every numeric List
// argument, so this layer only widens an Int list to Real (plotNumericValue)
// and dispatches to the matching runtime helper.
func (generator *generator) plotCall(value *ir.CallExpr) string {
	generator.usesPlot = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), plotModulePrefix)
	numeric := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, "Plot."+name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "nil"
		}
		return generator.plotNumericListValue(value.Arguments[index].Value, meta)
	}
	integer := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, "Plot."+name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "int64(0)"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	list := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, "Plot."+name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "nil"
		}
		return generator.expr(value.Arguments[index].Value)
	}
	switch name {
	case "new":
		return generator.plotChartFrom("AhdPlotNew()", meta)
	case "line":
		return generator.plotChartFrom("AhdPlotLine("+plotErrorRuntime+", "+numeric(0)+", "+numeric(1)+")", meta)
	case "scatter":
		return generator.plotChartFrom("AhdPlotScatter("+plotErrorRuntime+", "+numeric(0)+", "+numeric(1)+")", meta)
	case "bar":
		return generator.plotChartFrom("AhdPlotBar("+plotErrorRuntime+", "+list(0)+", "+numeric(1)+")", meta)
	case "histogram":
		return generator.plotChartFrom("AhdPlotHistogram("+plotErrorRuntime+", "+numeric(0)+", "+integer(1)+")", meta)
	case "box":
		return generator.plotChartFrom("AhdPlotBox("+plotErrorRuntime+", "+numeric(0)+")", meta)
	case "errorBar":
		return generator.plotChartFrom("AhdPlotErrorBar("+plotErrorRuntime+", "+numeric(0)+", "+numeric(1)+", "+numeric(2)+", "+numeric(3)+")", meta)
	case "subplots":
		return generator.plotSubplots(value, meta)
	default:
		return generator.unsupported("Plot function "+name, meta.Span)
	}
}

// plotNumericListValue renders one List<Int>|List<Real> argument as Go code
// of type *AhdList[float64], widening a List<Int> at the call site so every
// Plot rendering helper works on one canonical numeric representation.
func (generator *generator) plotNumericListValue(expression ir.Expr, meta ir.ExprBase) string {
	element := ir.Type{Kind: ir.InvalidType}
	if listType := expression.ExprMeta().Type; listType.Kind == ir.ListType && listType.Element != nil {
		element = *listType.Element
	}
	rendered := generator.expr(expression)
	switch element.Kind {
	case ir.RealType:
		return rendered
	case ir.IntType:
		return "AhdPlotWidenList(" + rendered + ")"
	default:
		if expression.ExprMeta().Type.Kind == ir.ClassType && expression.ExprMeta().Type.Class == numericVectorClass {
			return generator.numericVectorOf(expression) + ".Values"
		}
		return generator.unsupported("Plot numeric argument over "+expression.ExprMeta().Type.String(), meta.Span)
	}
}

// plotSubplots lowers Plot.subplots. rows and columns are each evaluated
// once, through a lambda, since AhdPlotSubplotsValidate and the generated
// Figure constructor both need them.
func (generator *generator) plotSubplots(value *ir.CallExpr, meta ir.ExprBase) string {
	if len(value.Arguments) != 3 || value.Arguments[0].Value == nil ||
		value.Arguments[1].Value == nil || value.Arguments[2].Value == nil {
		generator.fail(CodeGenerationFailure, "Plot.subplots has a missing argument", meta.Span, "the IR call is malformed")
		return "nil"
	}
	figureLayout := generator.layouts[plotFigureClass]
	if figureLayout == nil {
		return generator.unsupported("a Figure value without its Class declaration", meta.Span)
	}
	constructor := generator.functions[figureLayout.class.Constructor]
	if constructor == nil {
		return generator.unsupported("a Figure value without its Class declaration", meta.Span)
	}
	rows := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false)
	columns := generator.value(value.Arguments[1].Value, ir.Type{Kind: ir.IntType}, false)
	charts := generator.expr(value.Arguments[2].Value)
	chartsType := "*AhdList[" + generator.interfaceName(plotChartClass) + "]"
	return "func(rows int64, columns int64, charts " + chartsType + ") " + generator.interfaceName(plotFigureClass) + " { " +
		"return " + generator.callableName(constructor) + "(rows, columns, AhdPlotSubplotsValidate(" +
		plotErrorRuntime + ", rows, columns, charts)) }(" + rows + ", " + columns + ", " + charts + ")"
}

// plotChartFrom wraps a runtime AhdChart-producing expression into a real
// Chart value, via the generated all-fields constructor.
func (generator *generator) plotChartFrom(chart string, meta ir.ExprBase) string {
	helper, ok := generator.plotChartHelper()
	if !ok {
		return generator.unsupported("a Chart value without its Class declaration", meta.Span)
	}
	return helper + "(" + chart + ")"
}

// plotChartHelper registers (once per program) the generated wrapper that
// turns one AhdChart reading into a real Chart value, mirroring
// dataHelper/timeHelper's shared registration map.
func (generator *generator) plotChartHelper() (string, bool) {
	if generator.layouts[plotChartClass] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[plotChartClass]; known {
		return name, true
	}
	name := mangleNamed("ph_", generator.classDisplayName(plotChartClass), string(plotChartClass))
	generator.timeHelpers[plotChartClass] = name
	return name, true
}

// emitPlotHelpers writes the Chart wrapper, turning one AhdChart reading
// into a constructed AhdCode value.
func (generator *generator) emitPlotHelpers(writer *emitter) {
	name, known := generator.timeHelpers[plotChartClass]
	if !known {
		return
	}
	layout := generator.layouts[plotChartClass]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	arguments := make([]string, len(plotFields))
	for index, field := range plotFields {
		arguments[index] = "chart." + plotChartInterchangeField(field)
	}
	writer.line("// Chart value built from one runtime chart reading.")
	writer.open("func " + name + "(chart AhdChart) " + generator.interfaceName(plotChartClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(" + strings.Join(arguments, ", ") + ")")
	writer.close("}")
	writer.blank()
}

// plotChartOfLiteral builds an AhdChart composite literal reading every
// hidden storage field through its generated getter, given a receiver
// expression that is already safe to reference multiple times (a bound
// variable name -- never a raw, possibly side-effecting expression).
func (generator *generator) plotChartOfLiteral(receiver string) string {
	parts := make([]string, len(plotFields))
	for index, field := range plotFields {
		parts[index] = plotChartInterchangeField(field) + ": " +
			receiver + "." + generator.fieldName(plotFieldID(field)) + "_get()"
	}
	return "AhdChart{" + strings.Join(parts, ", ") + "}"
}

// plotChartOf evaluates one Chart expression exactly once and reads its
// hidden storage fields into the runtime interchange shape, mirroring
// tableOf: the expression is bound through a lambda parameter, since
// plotChartOfLiteral references it once per field (twenty times).
func (generator *generator) plotChartOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	return "func(value " + generator.interfaceName(plotChartClass) + ") AhdChart { return " +
		generator.plotChartOfLiteral("value") + " }(" + rendered + ")"
}

// plotChartInterchangeField maps one Chart storage field's AhdCode name onto
// its AhdChart interchange struct field name.
func plotChartInterchangeField(name string) string {
	switch name {
	case "kind":
		return "Kind"
	case "seriesKinds":
		return "SeriesKinds"
	case "seriesLabels":
		return "SeriesLabels"
	case "seriesX":
		return "SeriesX"
	case "seriesY":
		return "SeriesY"
	case "barLabels":
		return "BarLabels"
	case "barValues":
		return "BarValues"
	case "histogramValues":
		return "HistogramValues"
	case "histogramBins":
		return "HistogramBins"
	case "boxValues":
		return "BoxValues"
	case "errorX":
		return "ErrorX"
	case "errorY":
		return "ErrorY"
	case "errorLower":
		return "ErrorLower"
	case "errorUpper":
		return "ErrorUpper"
	case "title":
		return "Title"
	case "xLabel":
		return "XLabel"
	case "yLabel":
		return "YLabel"
	case "legend":
		return "Legend"
	case "width":
		return "Width"
	case "height":
		return "Height"
	default:
		return name
	}
}

// plotOperation lowers the built-in members of Chart and Figure.
func (generator *generator) plotOperation(name string, value *ir.CallExpr) string {
	generator.usesPlot = true
	meta := value.ExprMeta()
	if value.Callee == nil {
		generator.fail(CodeGenerationFailure, name+" has no receiver", meta.Span, "the IR call is malformed")
		return "nil"
	}
	text := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return `""`
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	integer := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "int64(0)"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	boolean := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "false"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}

	switch name {
	case "Chart.title":
		chart := generator.plotChartOf(value.Callee)
		return generator.plotChartFrom("AhdPlotChartTitle("+chart+", "+text(0)+")", meta)
	case "Chart.xLabel":
		chart := generator.plotChartOf(value.Callee)
		return generator.plotChartFrom("AhdPlotChartXLabel("+chart+", "+text(0)+")", meta)
	case "Chart.yLabel":
		chart := generator.plotChartOf(value.Callee)
		return generator.plotChartFrom("AhdPlotChartYLabel("+chart+", "+text(0)+")", meta)
	case "Chart.legend":
		chart := generator.plotChartOf(value.Callee)
		return generator.plotChartFrom("AhdPlotChartLegend("+chart+", "+boolean(0)+")", meta)
	case "Chart.size":
		chart := generator.plotChartOf(value.Callee)
		return generator.plotChartFrom("AhdPlotChartSize("+plotErrorRuntime+", "+chart+", "+integer(0)+", "+integer(1)+")", meta)
	case "Chart.line", "Chart.scatter":
		return generator.plotAddSeries(name, value, meta)
	case "Chart.save":
		chart := generator.plotChartOf(value.Callee)
		return "AhdPlotChartSave(" + plotErrorRuntime + ", " + chart + ", " + text(0) + ")"
	case "Chart.show":
		chart := generator.plotChartOf(value.Callee)
		return "AhdPlotChartShow(" + plotErrorRuntime + ", " + chart + ")"
	case "Figure.save":
		return generator.plotFigureOperation(value, text(0), meta, false)
	case "Figure.show":
		return generator.plotFigureOperation(value, "", meta, true)
	default:
		return generator.unsupported("Plot operation "+name, meta.Span)
	}
}

func (generator *generator) plotAddSeries(name string, value *ir.CallExpr, meta ir.ExprBase) string {
	kind := "line"
	if name == "Chart.scatter" {
		kind = "scatter"
	}
	if len(value.Arguments) != 3 || value.Arguments[0].Value == nil ||
		value.Arguments[1].Value == nil || value.Arguments[2].Value == nil {
		generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
		return "nil"
	}
	chart := generator.plotChartOf(value.Callee)
	x := generator.plotNumericListValue(value.Arguments[0].Value, meta)
	y := generator.plotNumericListValue(value.Arguments[1].Value, meta)
	label := generator.value(value.Arguments[2].Value, ir.Type{Kind: ir.StringType}, false)
	return generator.plotChartFrom("AhdPlotChartAddSeries("+plotErrorRuntime+", "+chart+", "+
		`"`+kind+`", `+x+", "+y+", "+label+")", meta)
}

// plotFigureOperation lowers Figure.save and Figure.show. Rows, columns, and
// the receiver are each evaluated once through a lambda, then every present
// Chart cell is converted to AhdChart via AhdPlotChartsFrom, matching
// plotChartOf's field-getter reads.
func (generator *generator) plotFigureOperation(value *ir.CallExpr, path string, meta ir.ExprBase, show bool) string {
	receiver := generator.expr(value.Callee)
	rowsField := "figure." + generator.fieldName(plotFigureFieldID("rows")) + "_get()"
	columnsField := "figure." + generator.fieldName(plotFigureFieldID("columns")) + "_get()"
	chartsField := "figure." + generator.fieldName(plotFigureFieldID("charts")) + "_get()"
	closureParameter := "v " + generator.interfaceName(plotChartClass)
	converted := "AhdPlotChartsFrom(" + chartsField + ", func(" + closureParameter + ") AhdChart { return " +
		generator.plotChartOfLiteral("v") + " })"
	if show {
		return "func(figure " + generator.interfaceName(plotFigureClass) + ") { " +
			"AhdPlotFigureShow(" + plotErrorRuntime + ", " + rowsField + ", " + columnsField + ", " + converted + ") " +
			"}(" + receiver + ")"
	}
	return "func(figure " + generator.interfaceName(plotFigureClass) + ") { " +
		"AhdPlotFigureSave(" + plotErrorRuntime + ", " + rowsField + ", " + columnsField + ", " + converted + ", " + path + ") " +
		"}(" + receiver + ")"
}
