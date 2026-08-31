package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// PlotModuleID is the synthetic module that carries the Plot standard
// library's Class declarations into the IR.
const PlotModuleID = "builtin:Plot"

const (
	plotChartClassID  = ir.ClassID(PlotModuleID + "::class::Chart")
	plotFigureClassID = ir.ClassID(PlotModuleID + "::class::Figure")
	plotErrorClassID  = ir.ClassID(PlotModuleID + "::class::PlotError")
)

func plotString() ir.Type { return ir.Type{Kind: ir.StringType} }
func plotInt() ir.Type    { return ir.Type{Kind: ir.IntType} }
func plotReal() ir.Type   { return ir.Type{Kind: ir.RealType} }
func plotBool() ir.Type   { return ir.Type{Kind: ir.BoolType} }

func plotList(element ir.Type) ir.Type {
	return ir.Type{Kind: ir.ListType, Element: &element}
}

func plotRealList() ir.Type   { return plotList(plotReal()) }
func plotStringList() ir.Type { return plotList(plotString()) }

// plotChartFields is the Chart storage layout: a kind discriminator plus one
// group of fields per chart family (line/scatter series, bar, histogram,
// box, errorBar), and the shared presentation metadata. Only the fields for
// the active kind hold data; this flat, kind-tagged shape mirrors Table's
// hidden-storage convention and keeps every field a plain List/String/
// Int/Real/Bool, with no variant/union type in the IR. Every field is
// Hidden, so has/has not report only PlotChartOperations.
func plotChartFields() []ir.Field {
	seriesX := plotList(plotRealList())
	seriesY := plotList(plotRealList())
	return []ir.Field{
		{ID: plotFieldID(plotChartClassID, "kind"), Name: "kind", Type: plotString(), NullState: ir.NonNull, Hidden: true},

		{ID: plotFieldID(plotChartClassID, "seriesKinds"), Name: "seriesKinds", Type: plotStringList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "seriesLabels"), Name: "seriesLabels", Type: plotStringList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "seriesX"), Name: "seriesX", Type: seriesX, NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "seriesY"), Name: "seriesY", Type: seriesY, NullState: ir.NonNull, Hidden: true},

		{ID: plotFieldID(plotChartClassID, "barLabels"), Name: "barLabels", Type: plotStringList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "barValues"), Name: "barValues", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},

		{ID: plotFieldID(plotChartClassID, "histogramValues"), Name: "histogramValues", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "histogramBins"), Name: "histogramBins", Type: plotInt(), NullState: ir.NonNull, Hidden: true},

		{ID: plotFieldID(plotChartClassID, "boxValues"), Name: "boxValues", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},

		{ID: plotFieldID(plotChartClassID, "errorX"), Name: "errorX", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "errorY"), Name: "errorY", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "errorLower"), Name: "errorLower", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "errorUpper"), Name: "errorUpper", Type: plotRealList(), NullState: ir.NonNull, Hidden: true},

		{ID: plotFieldID(plotChartClassID, "title"), Name: "title", Type: plotString(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "xLabel"), Name: "xLabel", Type: plotString(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "yLabel"), Name: "yLabel", Type: plotString(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "legend"), Name: "legend", Type: plotBool(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "width"), Name: "width", Type: plotInt(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotChartClassID, "height"), Name: "height", Type: plotInt(), NullState: ir.NonNull, Hidden: true},
	}
}

// plotFigureFields is the Figure storage layout: the subplot grid dimensions
// and the row-major List of Chart cells (nullable, so a grid with fewer
// charts than cells has real gaps rather than synthetic empty Charts).
func plotFigureFields() []ir.Field {
	chart := ir.Type{Kind: ir.ClassType, Class: plotChartClassID}
	return []ir.Field{
		{ID: plotFieldID(plotFigureClassID, "rows"), Name: "rows", Type: plotInt(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotFigureClassID, "columns"), Name: "columns", Type: plotInt(), NullState: ir.NonNull, Hidden: true},
		{ID: plotFieldID(plotFigureClassID, "charts"), Name: "charts", Type: plotList(chart), NullState: ir.NonNull, Hidden: true},
	}
}

func plotFieldID(class ir.ClassID, name string) ir.FieldID {
	return ir.FieldID(string(class) + "::field::" + name)
}

// plotModule emits the Chart, Figure, and PlotError Classes as ordinary IR
// classes, mirroring the Data/Regex pattern: builtin classes with a
// backend-only constructor that AhdCode source can never call directly,
// since Plot publishes no constructor for them.
func plotModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	chartClass := &ir.Class{
		ID: plotChartClassID, Symbol: ir.SymbolID(string(plotChartClassID) + "::symbol"),
		Name: "Chart", Operations: semantic.PlotChartOperations,
		Fields:      plotChartFields(),
		Constructor: plotAllFieldsConstructorID(plotChartClassID),
	}
	module.Classes = append(module.Classes, chartClass)
	module.Functions = append(module.Functions, plotAllFieldsConstructor(chartClass))

	figureClass := &ir.Class{
		ID: plotFigureClassID, Symbol: ir.SymbolID(string(plotFigureClassID) + "::symbol"),
		Name: "Figure", Operations: semantic.PlotFigureOperations,
		Fields:      plotFigureFields(),
		Constructor: plotAllFieldsConstructorID(plotFigureClassID),
	}
	module.Classes = append(module.Classes, figureClass)
	module.Functions = append(module.Functions, plotAllFieldsConstructor(figureClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: plotErrorClassID, Symbol: ir.SymbolID(string(plotErrorClassID) + "::symbol"),
		Name: "PlotError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(plotErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func plotAllFieldsConstructorID(class ir.ClassID) ir.CallableID {
	return ir.CallableID(string(class) + "::constructor::all-fields")
}

// plotAllFieldsConstructor builds the backend-only constructor that
// materializes one Chart or Figure from every one of its storage fields.
// AhdCode source cannot reach it: Plot publishes no constructor for Chart or
// Figure, so values come only from Plot's module functions and Chart
// methods, which build and validate this shape themselves.
func plotAllFieldsConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	function := &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature:  ir.Signature{Return: ir.Type{Kind: ir.NothingType}},
		ReturnNull: ir.NonNull,
	}
	var statements []ir.Statement
	for _, field := range class.Fields {
		parameter := ir.Parameter{
			ID: ir.SymbolID(string(id) + "::parameter::" + field.Name), Name: field.Name,
			Type: field.Type, NullState: ir.NonNull,
		}
		function.Signature.Parameters = append(function.Signature.Parameters,
			ir.ParameterType{Name: field.Name, Type: field.Type})
		function.Parameters = append(function.Parameters, parameter)
		statements = append(statements, &ir.AssignStmt{
			Target: ir.Target{
				Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{
					ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
					Symbol:   receiver,
				},
			},
			Value: &ir.LoadExpr{
				ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID,
			},
		})
	}
	function.Body = ir.Block{Statements: statements}
	return function
}
