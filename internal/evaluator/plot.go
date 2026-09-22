package evaluator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
	"ahdcode/internal/plotproto"
)

// The Plot standard module's REPL implementation. Chart and Figure are
// immutable, Table-style values: every Chart/Figure method returns a new
// value, and the receiver is never mutated. Rendering happens out-of-process
// in the bundled ahdplot helper (see internal/plotproto), because Gonum
// cannot be linked into ahdruntime.go, which is embedded verbatim into every
// natively-compiled AhdCode program. Field IDs below must match
// internal/lowering/plot_builtin.go exactly.

const (
	plotChartClassID  = ir.ClassID("builtin:Plot::class::Chart")
	plotFigureClassID = ir.ClassID("builtin:Plot::class::Figure")
)

var (
	plotFieldKind              = ir.FieldID(string(plotChartClassID) + "::field::kind")
	plotFieldSeriesKinds       = ir.FieldID(string(plotChartClassID) + "::field::seriesKinds")
	plotFieldSeriesLabels      = ir.FieldID(string(plotChartClassID) + "::field::seriesLabels")
	plotFieldSeriesX           = ir.FieldID(string(plotChartClassID) + "::field::seriesX")
	plotFieldSeriesY           = ir.FieldID(string(plotChartClassID) + "::field::seriesY")
	plotFieldSeriesLineStyles  = ir.FieldID(string(plotChartClassID) + "::field::seriesLineStyles")
	plotFieldSeriesLineWidths  = ir.FieldID(string(plotChartClassID) + "::field::seriesLineWidths")
	plotFieldSeriesMarkers     = ir.FieldID(string(plotChartClassID) + "::field::seriesMarkers")
	plotFieldSeriesMarkerSizes = ir.FieldID(string(plotChartClassID) + "::field::seriesMarkerSizes")
	plotFieldBarLabels         = ir.FieldID(string(plotChartClassID) + "::field::barLabels")
	plotFieldBarValues         = ir.FieldID(string(plotChartClassID) + "::field::barValues")
	plotFieldHistogramValues   = ir.FieldID(string(plotChartClassID) + "::field::histogramValues")
	plotFieldHistogramBins     = ir.FieldID(string(plotChartClassID) + "::field::histogramBins")
	plotFieldBoxValues         = ir.FieldID(string(plotChartClassID) + "::field::boxValues")
	plotFieldErrorX            = ir.FieldID(string(plotChartClassID) + "::field::errorX")
	plotFieldErrorY            = ir.FieldID(string(plotChartClassID) + "::field::errorY")
	plotFieldErrorLower        = ir.FieldID(string(plotChartClassID) + "::field::errorLower")
	plotFieldErrorUpper        = ir.FieldID(string(plotChartClassID) + "::field::errorUpper")
	plotFieldPieLabels         = ir.FieldID(string(plotChartClassID) + "::field::pieLabels")
	plotFieldPieValues         = ir.FieldID(string(plotChartClassID) + "::field::pieValues")
	plotFieldHeatmapXLabels    = ir.FieldID(string(plotChartClassID) + "::field::heatmapXLabels")
	plotFieldHeatmapYLabels    = ir.FieldID(string(plotChartClassID) + "::field::heatmapYLabels")
	plotFieldHeatmapValues     = ir.FieldID(string(plotChartClassID) + "::field::heatmapValues")
	plotFieldTitle             = ir.FieldID(string(plotChartClassID) + "::field::title")
	plotFieldXLabel            = ir.FieldID(string(plotChartClassID) + "::field::xLabel")
	plotFieldYLabel            = ir.FieldID(string(plotChartClassID) + "::field::yLabel")
	plotFieldLegend            = ir.FieldID(string(plotChartClassID) + "::field::legend")
	plotFieldLegendPosition    = ir.FieldID(string(plotChartClassID) + "::field::legendPosition")
	plotFieldWidth             = ir.FieldID(string(plotChartClassID) + "::field::width")
	plotFieldHeight            = ir.FieldID(string(plotChartClassID) + "::field::height")

	plotFigureFieldRows    = ir.FieldID(string(plotFigureClassID) + "::field::rows")
	plotFigureFieldColumns = ir.FieldID(string(plotFigureClassID) + "::field::columns")
	plotFigureFieldCharts  = ir.FieldID(string(plotFigureClassID) + "::field::charts")
)

const (
	plotDefaultWidth  = int64(800)
	plotDefaultHeight = int64(600)
)

// plotChart is the working shape one Chart operation reads and writes. Kind
// discriminates which of the family-specific fields are populated; the rest
// stay at their zero value, mirroring the Hidden field layout in
// internal/lowering/plot_builtin.go.
type plotChart struct {
	kind string

	seriesKinds       []string
	seriesLabels      []string
	seriesX           [][]float64
	seriesY           [][]float64
	seriesLineStyles  []string
	seriesLineWidths  []float64
	seriesMarkers     []string
	seriesMarkerSizes []float64

	barLabels []string
	barValues []float64

	histogramValues []float64
	histogramBins   int64

	boxValues []float64

	errorX, errorY, errorLower, errorUpper []float64

	pieLabels []string
	pieValues []float64

	heatmapXLabels, heatmapYLabels []string
	heatmapValues                  [][]float64

	title, xLabel, yLabel string
	legend                bool
	legendPosition        string
	width, height         int64
}

// chartOf reads a Chart instance's hidden storage fields. The stored Lists
// are never handed out, so a caller cannot reach them to mutate.
func (session *Session) chartOf(value any) plotChart {
	instance := session.requireInstance(value)
	field := func(id ir.FieldID) any { return instance.Fields[id] }
	return plotChart{
		kind:              field(plotFieldKind).(string),
		seriesKinds:       plotStringsFromField(field(plotFieldSeriesKinds)),
		seriesLabels:      plotStringsFromField(field(plotFieldSeriesLabels)),
		seriesX:           plotRealGridFromField(field(plotFieldSeriesX)),
		seriesY:           plotRealGridFromField(field(plotFieldSeriesY)),
		seriesLineStyles:  plotStringsFromField(field(plotFieldSeriesLineStyles)),
		seriesLineWidths:  plotRealsFromField(field(plotFieldSeriesLineWidths)),
		seriesMarkers:     plotStringsFromField(field(plotFieldSeriesMarkers)),
		seriesMarkerSizes: plotRealsFromField(field(plotFieldSeriesMarkerSizes)),
		barLabels:         plotStringsFromField(field(plotFieldBarLabels)),
		barValues:         plotRealsFromField(field(plotFieldBarValues)),
		histogramValues:   plotRealsFromField(field(plotFieldHistogramValues)),
		histogramBins:     field(plotFieldHistogramBins).(int64),
		boxValues:         plotRealsFromField(field(plotFieldBoxValues)),
		errorX:            plotRealsFromField(field(plotFieldErrorX)),
		errorY:            plotRealsFromField(field(plotFieldErrorY)),
		errorLower:        plotRealsFromField(field(plotFieldErrorLower)),
		errorUpper:        plotRealsFromField(field(plotFieldErrorUpper)),
		pieLabels:         plotStringsFromField(field(plotFieldPieLabels)),
		pieValues:         plotRealsFromField(field(plotFieldPieValues)),
		heatmapXLabels:    plotStringsFromField(field(plotFieldHeatmapXLabels)),
		heatmapYLabels:    plotStringsFromField(field(plotFieldHeatmapYLabels)),
		heatmapValues:     plotRealGridFromField(field(plotFieldHeatmapValues)),
		title:             field(plotFieldTitle).(string),
		xLabel:            field(plotFieldXLabel).(string),
		yLabel:            field(plotFieldYLabel).(string),
		legend:            field(plotFieldLegend).(bool),
		legendPosition:    field(plotFieldLegendPosition).(string),
		width:             field(plotFieldWidth).(int64),
		height:            field(plotFieldHeight).(int64),
	}
}

// plotChartValue materializes a validated chart as a new Chart instance.
func plotChartValue(chart plotChart) *Instance {
	return &Instance{Class: plotChartClassID, Fields: map[ir.FieldID]any{
		plotFieldKind:              chart.kind,
		plotFieldSeriesKinds:       plotStringsToField(chart.seriesKinds),
		plotFieldSeriesLabels:      plotStringsToField(chart.seriesLabels),
		plotFieldSeriesX:           plotRealGridToField(chart.seriesX),
		plotFieldSeriesY:           plotRealGridToField(chart.seriesY),
		plotFieldSeriesLineStyles:  plotStringsToField(chart.seriesLineStyles),
		plotFieldSeriesLineWidths:  plotRealsToField(chart.seriesLineWidths),
		plotFieldSeriesMarkers:     plotStringsToField(chart.seriesMarkers),
		plotFieldSeriesMarkerSizes: plotRealsToField(chart.seriesMarkerSizes),
		plotFieldBarLabels:         plotStringsToField(chart.barLabels),
		plotFieldBarValues:         plotRealsToField(chart.barValues),
		plotFieldHistogramValues:   plotRealsToField(chart.histogramValues),
		plotFieldHistogramBins:     chart.histogramBins,
		plotFieldBoxValues:         plotRealsToField(chart.boxValues),
		plotFieldErrorX:            plotRealsToField(chart.errorX),
		plotFieldErrorY:            plotRealsToField(chart.errorY),
		plotFieldErrorLower:        plotRealsToField(chart.errorLower),
		plotFieldErrorUpper:        plotRealsToField(chart.errorUpper),
		plotFieldPieLabels:         plotStringsToField(chart.pieLabels),
		plotFieldPieValues:         plotRealsToField(chart.pieValues),
		plotFieldHeatmapXLabels:    plotStringsToField(chart.heatmapXLabels),
		plotFieldHeatmapYLabels:    plotStringsToField(chart.heatmapYLabels),
		plotFieldHeatmapValues:     plotRealGridToField(chart.heatmapValues),
		plotFieldTitle:             chart.title,
		plotFieldXLabel:            chart.xLabel,
		plotFieldYLabel:            chart.yLabel,
		plotFieldLegend:            chart.legend,
		plotFieldLegendPosition:    chart.legendPosition,
		plotFieldWidth:             chart.width,
		plotFieldHeight:            chart.height,
	}}
}

func plotRealsFromField(value any) []float64 {
	list, _ := value.(*List)
	if list == nil {
		return nil
	}
	numbers := make([]float64, len(list.Items))
	for index, item := range list.Items {
		numbers[index] = item.(float64)
	}
	return numbers
}

func plotStringsFromField(value any) []string {
	list, _ := value.(*List)
	if list == nil {
		return nil
	}
	strings := make([]string, len(list.Items))
	for index, item := range list.Items {
		strings[index] = item.(string)
	}
	return strings
}

func plotRealGridFromField(value any) [][]float64 {
	list, _ := value.(*List)
	if list == nil {
		return nil
	}
	grid := make([][]float64, len(list.Items))
	for index, item := range list.Items {
		grid[index] = plotRealsFromField(item)
	}
	return grid
}

func plotRealsToField(values []float64) *List {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return &List{Items: items}
}

func plotStringsToField(values []string) *List {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return &List{Items: items}
}

func plotRealGridToField(grid [][]float64) *List {
	items := make([]any, len(grid))
	for index, row := range grid {
		items[index] = plotRealsToField(row)
	}
	return &List{Items: items}
}

// plotNumbers snapshots a List<Int> or List<Real> argument as float64, safely
// widening any Int elements, matching Statistics' internal Int -> Real
// convention.
func (session *Session) plotNumbers(value any) []float64 {
	if instance, ok := value.(*Instance); ok && instance != nil && instance.Class == evalVectorClass {
		return session.vectorValues(instance)
	}
	list := session.requireList(value)
	numbers := make([]float64, len(list.Items))
	for index, item := range list.Items {
		switch number := item.(type) {
		case int64:
			numbers[index] = float64(number)
		case float64:
			numbers[index] = number
		default:
			session.raise("NullError", "numeric List element is null")
		}
	}
	return numbers
}

func (session *Session) plotStrings(value any) []string {
	list := session.requireList(value)
	strings := make([]string, len(list.Items))
	for index, item := range list.Items {
		text, ok := item.(string)
		if !ok {
			session.raise("NullError", "String List element is null")
		}
		strings[index] = text
	}
	return strings
}

func (session *Session) plotRequireNonEmpty(count int, what string) {
	if count == 0 {
		session.raise("PlotError", what+" must not be empty")
	}
}

func (session *Session) plotRequireNonNegative(values []float64, what string) {
	for _, value := range values {
		if value < 0 {
			session.raise("PlotError", what+" must be non-negative")
		}
	}
}

// plotBuiltin dispatches one Plot module-level function.
func (session *Session) plotBuiltin(name string, arguments []any) any {
	switch name {
	case "new":
		return plotChartValue(plotChart{kind: "empty", legendPosition: "topRight", width: plotDefaultWidth, height: plotDefaultHeight})
	case "line":
		return session.plotNewSeries("line", arguments)
	case "scatter":
		return session.plotNewSeries("scatter", arguments)
	case "bar":
		labels := session.plotStrings(arguments[0])
		values := session.plotNumbers(arguments[1])
		if len(labels) != len(values) {
			session.raise("PlotError", "bar labels and values must have the same length")
		}
		session.plotRequireNonEmpty(len(values), "bar chart data")
		return plotChartValue(plotChart{kind: "bar", barLabels: labels, barValues: values, legendPosition: "topRight",
			width: plotDefaultWidth, height: plotDefaultHeight})
	case "pie":
		return session.plotPieBuiltin(arguments)
	case "heatmap":
		return session.plotHeatmapBuiltin(arguments)
	case "histogram":
		values := session.plotNumbers(arguments[0])
		bins := arguments[1].(int64)
		if bins <= 0 {
			session.raise("PlotError", "histogram bin count must be positive")
		}
		session.plotRequireNonEmpty(len(values), "histogram data")
		return plotChartValue(plotChart{kind: "histogram", histogramValues: values, histogramBins: bins, legendPosition: "topRight",
			width: plotDefaultWidth, height: plotDefaultHeight})
	case "box":
		values := session.plotNumbers(arguments[0])
		session.plotRequireNonEmpty(len(values), "box plot data")
		return plotChartValue(plotChart{kind: "box", boxValues: values, legendPosition: "topRight",
			width: plotDefaultWidth, height: plotDefaultHeight})
	case "errorBar":
		x := session.plotNumbers(arguments[0])
		y := session.plotNumbers(arguments[1])
		lower := session.plotNumbers(arguments[2])
		upper := session.plotNumbers(arguments[3])
		if len(x) != len(y) || len(y) != len(lower) || len(lower) != len(upper) {
			session.raise("PlotError", "errorBar x, y, lowerErrors, and upperErrors must have the same length")
		}
		session.plotRequireNonEmpty(len(x), "errorBar data")
		session.plotRequireNonNegative(lower, "lowerErrors")
		session.plotRequireNonNegative(upper, "upperErrors")
		return plotChartValue(plotChart{kind: "errorBar", errorX: x, errorY: y, errorLower: lower, errorUpper: upper, legendPosition: "topRight",
			width: plotDefaultWidth, height: plotDefaultHeight})
	case "surface":
		return session.plotSurfaceBuiltin(arguments)
	case "subplots":
		rows, columns := arguments[0].(int64), arguments[1].(int64)
		if rows <= 0 || columns <= 0 {
			session.raise("PlotError", "subplot rows and columns must be positive")
		}
		list := session.requireList(arguments[2])
		if int64(len(list.Items)) > rows*columns {
			session.raise("PlotError", "more charts than subplot cells")
		}
		charts := make([]*Instance, len(list.Items))
		for index, item := range list.Items {
			charts[index] = session.requireInstance(item)
		}
		return plotFigureValue(rows, columns, charts)
	}
	session.raise("Error", "unsupported Plot function "+name)
	return nil
}

func (session *Session) plotNewSeries(kind string, arguments []any) any {
	x := session.plotNumbers(arguments[0])
	y := session.plotNumbers(arguments[1])
	if len(x) != len(y) {
		session.raise("PlotError", "x and y must have the same length")
	}
	session.plotRequireNonEmpty(len(x), kind+" chart data")
	return plotChartValue(plotChart{
		kind: "line-scatter", seriesKinds: []string{kind}, seriesLabels: []string{""},
		seriesX: [][]float64{x}, seriesY: [][]float64{y},
		seriesLineStyles: []string{"solid"}, seriesLineWidths: []float64{1},
		seriesMarkers: []string{plotDefaultMarker(kind)}, seriesMarkerSizes: []float64{5},
		legendPosition: "topRight",
		width:          plotDefaultWidth, height: plotDefaultHeight,
	})
}

func plotDefaultMarker(kind string) string {
	if kind == "scatter" {
		return "circle"
	}
	return "none"
}

func plotLineStyleValue(value string) bool {
	return value == "solid" || value == "dashed" || value == "dotted" || value == "dashDot"
}

func plotMarkerValue(value string) bool {
	return value == "none" || value == "circle" || value == "square" || value == "triangle" || value == "diamond" || value == "cross"
}

func plotLegendPositionValue(value string) bool {
	return value == "topRight" || value == "topLeft" || value == "bottomRight" || value == "bottomLeft"
}

func plotSeriesStrings(values []string, count int, fallback string) []string {
	result := append([]string(nil), values...)
	for len(result) < count {
		result = append(result, fallback)
	}
	return result
}

func plotSeriesMarkers(values, kinds []string) []string {
	result := append([]string(nil), values...)
	for len(result) < len(kinds) {
		result = append(result, plotDefaultMarker(kinds[len(result)]))
	}
	return result
}

func plotSeriesFloats(values []float64, count int, fallback float64) []float64 {
	result := append([]float64(nil), values...)
	for len(result) < count {
		result = append(result, fallback)
	}
	return result
}

func (session *Session) plotLastSeriesIndex(chart plotChart, operation string) int {
	if chart.kind != "line-scatter" || len(chart.seriesKinds) == 0 {
		session.raise("PlotError", operation+" requires a line or scatter series")
	}
	return len(chart.seriesKinds) - 1
}

// plotFigureValue materializes a validated Figure. charts is stored exactly
// as given -- a possibly-shorter-than-rows*columns snapshot -- so "fewer
// charts than cells" needs no padding: the remaining row-major cells are
// simply absent and rendered blank.
func plotFigureValue(rows, columns int64, charts []*Instance) *Instance {
	items := make([]any, len(charts))
	for index, chart := range charts {
		items[index] = chart
	}
	return &Instance{Class: plotFigureClassID, Fields: map[ir.FieldID]any{
		plotFigureFieldRows: rows, plotFigureFieldColumns: columns,
		plotFigureFieldCharts: &List{Items: items},
	}}
}

func (session *Session) figureOf(value any) (rows, columns int64, charts []*Instance) {
	instance := session.requireInstance(value)
	rows = instance.Fields[plotFigureFieldRows].(int64)
	columns = instance.Fields[plotFigureFieldColumns].(int64)
	list, _ := instance.Fields[plotFigureFieldCharts].(*List)
	if list != nil {
		charts = make([]*Instance, len(list.Items))
		for index, item := range list.Items {
			charts[index] = item.(*Instance)
		}
	}
	return
}

// plotOperation dispatches one Chart or Figure member. Every Chart operation
// is pure: it reads a snapshot and returns a new Chart, leaving the receiver
// untouched, matching the Data Table convention.
func (session *Session) plotOperation(name string, receiver any, arguments []any) any {
	switch name {
	case "Chart.title":
		chart := session.chartOf(receiver)
		chart.title = arguments[0].(string)
		return plotChartValue(chart)
	case "Chart.xLabel":
		chart := session.chartOf(receiver)
		session.plotRequireAxes(chart, "xLabel")
		chart.xLabel = arguments[0].(string)
		return plotChartValue(chart)
	case "Chart.yLabel":
		chart := session.chartOf(receiver)
		session.plotRequireAxes(chart, "yLabel")
		chart.yLabel = arguments[0].(string)
		return plotChartValue(chart)
	case "Chart.legend":
		chart := session.chartOf(receiver)
		chart.legend = arguments[0].(bool)
		return plotChartValue(chart)
	case "Chart.legendPosition":
		chart := session.chartOf(receiver)
		position := arguments[0].(string)
		if !plotLegendPositionValue(position) {
			session.raise("PlotError", "unknown legend position "+position)
		}
		chart.legendPosition = position
		return plotChartValue(chart)
	case "Chart.size":
		chart := session.chartOf(receiver)
		width, height := arguments[0].(int64), arguments[1].(int64)
		if width <= 0 || height <= 0 {
			session.raise("PlotError", "chart size must be positive")
		}
		chart.width, chart.height = width, height
		return plotChartValue(chart)
	case "Chart.lineStyle":
		chart := session.chartOf(receiver)
		index := session.plotLastSeriesIndex(chart, "lineStyle")
		if chart.seriesKinds[index] != "line" {
			session.raise("PlotError", "lineStyle applies only to line series")
		}
		style := arguments[0].(string)
		if !plotLineStyleValue(style) {
			session.raise("PlotError", "unknown line style "+style)
		}
		chart.seriesLineStyles = plotSeriesStrings(chart.seriesLineStyles, len(chart.seriesKinds), "solid")
		chart.seriesLineStyles[index] = style
		return plotChartValue(chart)
	case "Chart.lineWidth":
		chart := session.chartOf(receiver)
		index := session.plotLastSeriesIndex(chart, "lineWidth")
		if chart.seriesKinds[index] != "line" {
			session.raise("PlotError", "lineWidth applies only to line series")
		}
		width := arguments[0].(float64)
		if math.IsNaN(width) || math.IsInf(width, 0) || width <= 0 || width > 32 {
			session.raise("PlotError", "line width must be finite and between 0 and 32")
		}
		chart.seriesLineWidths = plotSeriesFloats(chart.seriesLineWidths, len(chart.seriesKinds), 1)
		chart.seriesLineWidths[index] = width
		return plotChartValue(chart)
	case "Chart.marker":
		chart := session.chartOf(receiver)
		index := session.plotLastSeriesIndex(chart, "marker")
		marker := arguments[0].(string)
		if !plotMarkerValue(marker) {
			session.raise("PlotError", "unknown marker "+marker)
		}
		chart.seriesMarkers = plotSeriesMarkers(chart.seriesMarkers, chart.seriesKinds)
		chart.seriesMarkers[index] = marker
		return plotChartValue(chart)
	case "Chart.markerSize":
		chart := session.chartOf(receiver)
		index := session.plotLastSeriesIndex(chart, "markerSize")
		size := arguments[0].(float64)
		if math.IsNaN(size) || math.IsInf(size, 0) || size <= 0 || size > 64 {
			session.raise("PlotError", "marker size must be finite and between 0 and 64")
		}
		chart.seriesMarkerSizes = plotSeriesFloats(chart.seriesMarkerSizes, len(chart.seriesKinds), 5)
		chart.seriesMarkerSizes[index] = size
		return plotChartValue(chart)
	case "Chart.line", "Chart.scatter":
		return session.plotAddSeries(name, receiver, arguments)
	case "Chart.save":
		return session.plotSaveChart(receiver, arguments[0].(string))
	case "Chart.show":
		return session.plotShowChart(receiver)
	case "Figure.save":
		return session.plotSaveFigure(receiver, arguments[0].(string))
	case "Figure.show":
		return session.plotShowFigure(receiver)
	}
	session.raise("Error", "unsupported Plot operation "+name)
	return nil
}

func (session *Session) plotAddSeries(name string, receiver any, arguments []any) any {
	kind := "line"
	if name == "Chart.scatter" {
		kind = "scatter"
	}
	chart := session.chartOf(receiver)
	if chart.kind != "empty" && chart.kind != "line-scatter" {
		session.raise("PlotError", "cannot add a "+kind+" series to a "+chart.kind+" chart")
	}
	x := session.plotNumbers(arguments[0])
	y := session.plotNumbers(arguments[1])
	if len(x) != len(y) {
		session.raise("PlotError", "x and y must have the same length")
	}
	session.plotRequireNonEmpty(len(x), kind+" chart data")
	label := arguments[2].(string)
	chart.kind = "line-scatter"
	chart.seriesKinds = append(append([]string(nil), chart.seriesKinds...), kind)
	chart.seriesLabels = append(append([]string(nil), chart.seriesLabels...), label)
	chart.seriesX = append(append([][]float64(nil), chart.seriesX...), x)
	chart.seriesY = append(append([][]float64(nil), chart.seriesY...), y)
	chart.seriesLineStyles = append(append([]string(nil), chart.seriesLineStyles...), "solid")
	chart.seriesLineWidths = append(append([]float64(nil), chart.seriesLineWidths...), 1)
	chart.seriesMarkers = append(append([]string(nil), chart.seriesMarkers...), plotDefaultMarker(kind))
	chart.seriesMarkerSizes = append(append([]float64(nil), chart.seriesMarkerSizes...), 5)
	return plotChartValue(chart)
}

func (session *Session) plotSaveChart(receiver any, path string) any {
	chart := session.chartOf(receiver)
	resolved := session.sessionPath(path)
	session.plotRenderRequest(plotproto.Request{
		OutputPath: resolved, Width: int(chart.width), Height: int(chart.height),
		Rows: 1, Columns: 1, Charts: []plotproto.ChartSpec{plotChartSpec(chart)},
	})
	return Nothing
}

// plotShowChart renders the Chart exactly as save() would and opens it in
// AhdCode's interactive viewer, through the same runtime native programs use.
func (session *Session) plotShowChart(receiver any) any {
	chart := session.chartOf(receiver)
	session.plotCheck(ahdruntime.AhdPlotShowSizeProblem(chart.width, chart.height))
	path := session.plotTempImagePath()
	defer os.Remove(path)
	request := plotproto.Request{
		OutputPath: path, Width: int(chart.width), Height: int(chart.height),
		Rows: 1, Columns: 1, Charts: []plotproto.ChartSpec{plotChartSpec(chart)},
	}
	session.plotRenderRequest(request)
	request.OutputPath = ""
	session.plotCheck(ahdruntime.AhdPlotView(path, chart.title, request))
	return Nothing
}

func (session *Session) plotSaveFigure(receiver any, path string) any {
	rows, columns, charts := session.figureOf(receiver)
	resolved := session.sessionPath(path)
	width, height := plotFigureDefaultSize(rows, columns)
	session.plotRenderRequest(session.plotFigureRequest(rows, columns, charts, resolved, width, height))
	return Nothing
}

// plotShowFigure renders the Figure as one image and opens it in the viewer.
func (session *Session) plotShowFigure(receiver any) any {
	rows, columns, charts := session.figureOf(receiver)
	width, height := plotFigureDefaultSize(rows, columns)
	session.plotCheck(ahdruntime.AhdPlotShowSizeProblem(int64(width), int64(height)))
	path := session.plotTempImagePath()
	defer os.Remove(path)
	request := session.plotFigureRequest(rows, columns, charts, path, width, height)
	session.plotRenderRequest(request)
	request.OutputPath = ""
	session.plotCheck(ahdruntime.AhdPlotView(path, "", request))
	return Nothing
}

// plotFigureDefaultSize is Figure's one deterministic size rule: a fixed
// per-cell budget scaled by the grid dimensions. v0.1.14 does not publish a
// Figure.size method, keeping subplot sizing to this one predictable
// formula rather than a second configuration surface.
func plotFigureDefaultSize(rows, columns int64) (int, int) {
	return int(columns) * 500, int(rows) * 400
}

func (session *Session) plotFigureRequest(rows, columns int64, charts []*Instance, outputPath string, width, height int) plotproto.Request {
	cells := make([]plotproto.ChartSpec, rows*columns)
	for index := range cells {
		if int64(index) < int64(len(charts)) {
			cells[index] = plotChartSpec(session.chartOf(charts[index]))
		}
	}
	return plotproto.Request{OutputPath: outputPath, Width: width, Height: height, Rows: int(rows), Columns: int(columns), Charts: cells}
}

func plotChartSpec(chart plotChart) plotproto.ChartSpec {
	spec := plotproto.ChartSpec{
		Present: true, Kind: chart.kind,
		Title: chart.title, XLabel: chart.xLabel, YLabel: chart.yLabel, Legend: chart.legend,
		LegendPosition: chart.legendPosition,
	}
	switch chart.kind {
	case "line-scatter":
		for index := range chart.seriesKinds {
			spec.Series = append(spec.Series, plotproto.SeriesSpec{
				Kind: chart.seriesKinds[index], Label: chart.seriesLabels[index],
				X: chart.seriesX[index], Y: chart.seriesY[index],
				LineStyle:  plotSeriesStringAt(chart.seriesLineStyles, index, "solid"),
				LineWidth:  plotSeriesFloatAt(chart.seriesLineWidths, index, 1),
				Marker:     plotSeriesStringAt(chart.seriesMarkers, index, plotDefaultMarker(chart.seriesKinds[index])),
				MarkerSize: plotSeriesFloatAt(chart.seriesMarkerSizes, index, 5),
			})
		}
	case "bar":
		spec.BarLabels, spec.BarValues = chart.barLabels, chart.barValues
	case "histogram":
		spec.HistogramValues, spec.HistogramBins = chart.histogramValues, int(chart.histogramBins)
	case "box":
		spec.BoxValues = chart.boxValues
	case "errorBar":
		spec.ErrorX, spec.ErrorY, spec.ErrorLower, spec.ErrorUpper = chart.errorX, chart.errorY, chart.errorLower, chart.errorUpper
	case "pie":
		spec.PieLabels, spec.PieValues = chart.pieLabels, chart.pieValues
	case "heatmap":
		spec.HeatmapXLabels, spec.HeatmapYLabels = chart.heatmapXLabels, chart.heatmapYLabels
		spec.HeatmapValues = chart.heatmapValues
	}
	return spec
}

func plotSeriesStringAt(values []string, index int, fallback string) string {
	if index >= 0 && index < len(values) && values[index] != "" {
		return values[index]
	}
	return fallback
}

func plotSeriesFloatAt(values []float64, index int, fallback float64) float64 {
	if index >= 0 && index < len(values) && values[index] > 0 {
		return values[index]
	}
	return fallback
}

// plotTempDir is AhdCode's own temporary area for Chart.show/Figure.show
// preview images, kept separate from the system temp root's general clutter.
func plotTempDir() (string, error) {
	dir := filepath.Join(os.TempDir(), "ahdcode", "plot")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func (session *Session) plotTempImagePath() string {
	dir, err := plotTempDir()
	if err != nil {
		session.raise("PlotError", "creating temporary directory: "+err.Error())
	}
	file, err := os.CreateTemp(dir, "chart-*.png")
	if err != nil {
		session.raise("PlotError", "creating temporary file: "+err.Error())
	}
	path := file.Name()
	file.Close()
	return path
}

// discoverPlotRuntime locates the bundled ahdplot renderer helper: an
// explicit override, then a path relative to the running executable,
// mirroring the Latex module's ahdLatexRuntime discovery.
func discoverPlotRuntime() (string, error) {
	if custom := os.Getenv("AHDCODE_PLOT_RUNTIME"); custom != "" {
		return custom, nil
	}
	if bin, ok := interpreterBinDirectory(); ok {
		candidates := []string{
			filepath.Join(bin, "ahdplot"),
			filepath.Join(bin, "..", "libexec", "ahdcode", "ahdplot"),
		}
		if runtime.GOOS == "windows" {
			candidates = append([]string{filepath.Join(bin, "ahdplot.exe")}, candidates...)
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", errors.New("the Plot renderer helper (ahdplot) was not found; set AHDCODE_PLOT_RUNTIME " +
		"or reinstall AhdCode with the bundled Plot renderer")
}

// plotRenderRequest hands one render request to the ahdplot helper: write it
// to a temporary file, run the helper against it, and require its Response
// to report success. Every failure path -- missing helper, timeout,
// malformed response, renderer-reported error -- becomes a PlotError, never
// a leaked Go/filesystem error.
func (session *Session) plotRenderRequest(request plotproto.Request) {
	runtimePath, err := discoverPlotRuntime()
	if err != nil {
		session.raise("PlotError", err.Error())
	}
	dir, err := plotTempDir()
	if err != nil {
		session.raise("PlotError", "creating temporary directory: "+err.Error())
	}
	requestFile, err := os.CreateTemp(dir, "request-*.json")
	if err != nil {
		session.raise("PlotError", "writing render request: "+err.Error())
	}
	defer os.Remove(requestFile.Name())
	encoded, err := json.Marshal(request)
	if err == nil {
		_, err = requestFile.Write(encoded)
	}
	closeErr := requestFile.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		session.raise("PlotError", "writing render request: "+err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, runtimePath, requestFile.Name()).Output()
	var response plotproto.Response
	_ = json.Unmarshal(output, &response)
	if runErr != nil || !response.OK {
		message := response.Message
		if message == "" && runErr != nil {
			message = runErr.Error()
		}
		session.raise("PlotError", "rendering chart: "+message)
	}
}

// plotCheck raises a PlotError for a Plot runtime problem.
func (session *Session) plotCheck(problem string) {
	if problem != "" {
		session.raise("PlotError", problem)
	}
}

// plotPieBuiltin is Plot.pie(labels, values), checked exactly as the
// compiled runtime checks it.
func (session *Session) plotPieBuiltin(arguments []any) any {
	labels := session.plotStrings(arguments[0])
	values := session.plotNumbers(arguments[1])
	if len(labels) != len(values) {
		session.raise("PlotError", "pie labels and values must have the same length")
	}
	session.plotRequireNonEmpty(len(values), "pie chart data")
	if len(values) > ahdruntime.AhdPlotMaxPieSlices {
		session.raise("PlotError", fmt.Sprintf("a pie chart draws at most %d slices; got %d",
			ahdruntime.AhdPlotMaxPieSlices, len(values)))
	}
	session.plotRequireNonNegative(values, "pie values")
	positive := false
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			session.raise("PlotError", "pie values must be finite")
		}
		if value > 0 {
			positive = true
		}
	}
	if !positive {
		session.raise("PlotError", "a pie chart needs at least one value greater than zero")
	}
	// A pie's legend is on by default: its colours mean nothing without the
	// category names beside them.
	return plotChartValue(plotChart{kind: "pie", pieLabels: labels, pieValues: values,
		legend: true, legendPosition: "topRight", width: plotDefaultWidth, height: plotDefaultHeight})
}

// plotHeatmapBuiltin is Plot.heatmap(xLabels, yLabels, values).
func (session *Session) plotHeatmapBuiltin(arguments []any) any {
	xLabels := session.plotStrings(arguments[0])
	yLabels := session.plotStrings(arguments[1])
	values := session.matrixRows(arguments[2])
	session.plotRequireNonEmpty(len(xLabels), "heatmap x labels")
	session.plotRequireNonEmpty(len(yLabels), "heatmap y labels")
	if len(xLabels) > ahdruntime.AhdPlotMaxHeatmapLabels || len(yLabels) > ahdruntime.AhdPlotMaxHeatmapLabels {
		session.raise("PlotError", fmt.Sprintf("a heatmap has at most %d labels on each axis; got %d and %d",
			ahdruntime.AhdPlotMaxHeatmapLabels, len(xLabels), len(yLabels)))
	}
	if len(xLabels)*len(yLabels) > ahdruntime.AhdPlotMaxHeatmapCells {
		session.raise("PlotError", fmt.Sprintf("a heatmap draws at most %d cells; got %d",
			ahdruntime.AhdPlotMaxHeatmapCells, len(xLabels)*len(yLabels)))
	}
	if len(values) != len(yLabels) {
		session.raise("PlotError", fmt.Sprintf("the values Matrix has %d rows; a heatmap needs one row per y label (%d)",
			len(values), len(yLabels)))
	}
	for _, row := range values {
		if len(row) != len(xLabels) {
			session.raise("PlotError", fmt.Sprintf("the values Matrix has %d columns; a heatmap needs one column per x label (%d)",
				len(row), len(xLabels)))
		}
		for _, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				session.raise("PlotError", "heatmap values must be finite; NaN and Infinity cannot be drawn")
			}
		}
	}
	return plotChartValue(plotChart{kind: "heatmap", heatmapXLabels: xLabels, heatmapYLabels: yLabels,
		heatmapValues: values, legend: true, legendPosition: "topRight", width: plotDefaultWidth, height: plotDefaultHeight})
}

// plotRequireAxes refuses an axis title on a chart that has no axes.
func (session *Session) plotRequireAxes(chart plotChart, member string) {
	if chart.kind == "pie" {
		session.raise("PlotError", "a pie chart has no x or y axis, so "+member+" cannot be set on one")
	}
}
