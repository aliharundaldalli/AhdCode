// Command ahdplot is the bundled rendering helper for AhdCode's Plot
// standard module. It is a separate executable, not a library, because
// Gonum's plotting packages cannot be linked into ahdruntime.go: that file
// is embedded verbatim into every natively-compiled AhdCode program, which
// builds in an isolated, dependency-free workspace (see
// internal/build/pipeline.go and internal/plotproto). Both the native
// runtime and the persistent evaluator invoke this helper the same way:
//
//	ahdplot <request-file>
//
// The request file holds one JSON-encoded plotproto.Request. ahdplot renders
// it with Gonum and writes the image to Request.OutputPath, then writes one
// line of JSON (a plotproto.Response) to stdout and exits 0 on success, or
// exits nonzero with a Response{OK: false} describing the failure.
package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"

	"ahdcode/internal/plotproto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"

	_ "gonum.org/v1/plot/vg/vgimg"
	_ "gonum.org/v1/plot/vg/vgpdf"
	_ "gonum.org/v1/plot/vg/vgsvg"
)

func main() {
	if err := run(); err != nil {
		respond(false, err.Error())
		os.Exit(1)
	}
	respond(true, "")
}

func respond(ok bool, message string) {
	encoded, _ := json.Marshal(plotproto.Response{OK: ok, Message: message})
	fmt.Println(string(encoded))
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: ahdplot <request-file>")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		return fmt.Errorf("reading request: %w", err)
	}
	var request plotproto.Request
	if err := json.Unmarshal(raw, &request); err != nil {
		return fmt.Errorf("decoding request: %w", err)
	}
	return render(request)
}

func render(request plotproto.Request) error {
	if request.Mode == "math" {
		return renderMath(request)
	}
	if request.Rows <= 0 || request.Columns <= 0 {
		return fmt.Errorf("invalid grid dimensions %dx%d", request.Rows, request.Columns)
	}
	if len(request.Charts) != request.Rows*request.Columns {
		return fmt.Errorf("expected %d chart cells, got %d", request.Rows*request.Columns, len(request.Charts))
	}
	format, err := outputFormat(request.OutputPath)
	if err != nil {
		return err
	}
	width, height := vg.Points(float64(request.Width)), vg.Points(float64(request.Height))

	if request.Rows == 1 && request.Columns == 1 {
		chart := request.Charts[0]
		if !chart.Present {
			return fmt.Errorf("chart is empty: nothing to render")
		}
		p, bar, err := buildPlot(chart)
		if err != nil {
			return err
		}
		if bar == nil {
			if err := savePlot(p, request.OutputPath, format, width, height); err != nil {
				return fmt.Errorf("saving %s: %w", format, err)
			}
			return nil
		}
		return saveWithColorBar(p, bar, request.OutputPath, format, width, height)
	}

	return renderGrid(request, format, width, height)
}

// renderGrid composes a rows x columns Figure. An absent cell (fewer charts
// than grid cells) is left blank: its Plot has no title, labels, or
// plotters, matching the documented "blank remaining cells" subplot policy.
func renderGrid(request plotproto.Request, format string, width, height vg.Length) error {
	plots := make([][]*plot.Plot, request.Rows)
	bars := make([][]*plot.Plot, request.Rows)
	for row := 0; row < request.Rows; row++ {
		plots[row] = make([]*plot.Plot, request.Columns)
		bars[row] = make([]*plot.Plot, request.Columns)
		for column := 0; column < request.Columns; column++ {
			spec := request.Charts[row*request.Columns+column]
			if !spec.Present {
				plots[row][column] = plot.New()
				plots[row][column].HideAxes()
				continue
			}
			p, bar, err := buildPlot(spec)
			if err != nil {
				return err
			}
			plots[row][column], bars[row][column] = p, bar
		}
	}

	canvasWriter, err := draw.NewFormattedCanvas(width, height, format)
	if err != nil {
		return fmt.Errorf("preparing %s canvas: %w", format, err)
	}
	tiles := draw.Tiles{
		Rows: request.Rows, Cols: request.Columns,
		PadX: vg.Points(12), PadY: vg.Points(12),
		PadTop: vg.Points(8), PadBottom: vg.Points(8), PadLeft: vg.Points(8), PadRight: vg.Points(8),
	}
	canvas := draw.New(canvasWriter)
	aligned := plot.Align(plots, tiles, canvas)
	for row := 0; row < request.Rows; row++ {
		for column := 0; column < request.Columns; column++ {
			cell := aligned[row][column]
			if bar := bars[row][column]; bar != nil {
				grid, scale := splitForColorBar(cell)
				da := plots[row][column].DataCanvas(grid)
				plotTextHandler().titleShiftX = da.Center().X - grid.Center().X
				plots[row][column].Draw(grid)
				plotTextHandler().titleShiftX = 0
				bar.Draw(scale)
				continue
			}
			da := plots[row][column].DataCanvas(cell)
			plotTextHandler().titleShiftX = da.Center().X - cell.Center().X
			plots[row][column].Draw(cell)
			plotTextHandler().titleShiftX = 0
		}
	}

	file, err := os.Create(request.OutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()
	if _, err := canvasWriter.WriteTo(file); err != nil {
		return fmt.Errorf("writing %s: %w", format, err)
	}
	return nil
}

func outputFormat(path string) (string, error) {
	switch {
	case hasSuffix(path, ".png"):
		return "png", nil
	case hasSuffix(path, ".svg"):
		return "svg", nil
	case hasSuffix(path, ".pdf"):
		return "pdf", nil
	default:
		return "", fmt.Errorf("unsupported output format for %q; supported formats are .png, .svg, and .pdf", path)
	}
}

func hasSuffix(path, suffix string) bool {
	if len(path) < len(suffix) {
		return false
	}
	tail := path[len(path)-len(suffix):]
	if len(tail) != len(suffix) {
		return false
	}
	for i := range tail {
		a, b := tail[i], suffix[i]
		if 'A' <= a && a <= 'Z' {
			a += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

// buildPlot turns one chart spec into its plot. A heatmap with its legend
// on also produces a second plot -- the colour scale -- which the caller
// draws in a strip beside the first; every other chart returns nil for it.
func buildPlot(spec plotproto.ChartSpec) (*plot.Plot, *plot.Plot, error) {
	if err := validateChartMath(spec, plotTextHandler()); err != nil {
		return nil, nil, err
	}
	if err := validateLegendPosition(spec.LegendPosition); err != nil {
		return nil, nil, err
	}
	p := plot.New()
	configurePlotLayout(p, spec)
	p.Title.Text = spec.Title
	p.X.Label.Text = spec.XLabel
	p.Y.Label.Text = spec.YLabel

	switch spec.Kind {
	case "empty":
		// A blank canvas with only its metadata: Plot.new() with no series
		// added yet.
	case "line-scatter":
		if err := addSeries(p, spec); err != nil {
			return nil, nil, err
		}
	case "bar":
		if err := addBar(p, spec); err != nil {
			return nil, nil, err
		}
	case "histogram":
		if err := addHistogram(p, spec); err != nil {
			return nil, nil, err
		}
	case "box":
		if err := addBox(p, spec); err != nil {
			return nil, nil, err
		}
	case "errorBar":
		if err := addErrorBar(p, spec); err != nil {
			return nil, nil, err
		}
	case "pie":
		if err := addPie(p, spec); err != nil {
			return nil, nil, err
		}
	case "heatmap":
		bar, err := addHeatmap(p, spec)
		if err != nil {
			return nil, nil, err
		}
		return p, bar, nil
	default:
		return nil, nil, fmt.Errorf("unsupported chart kind %q", spec.Kind)
	}
	return p, nil, nil
}

const defaultLegendPosition = "topRight"

func validateLegendPosition(position string) error {
	if position == "" {
		return nil
	}
	switch position {
	case "topRight", "topLeft", "bottomRight", "bottomLeft":
		return nil
	default:
		return fmt.Errorf("unknown legend position %q", position)
	}
}

// configurePlotLayout keeps labels and the legend away from the chart edges.
// Gonum's default legend is bottom-right with zero offsets, which is a poor
// default for mathematical labels and dense academic plots.
func configurePlotLayout(p *plot.Plot, spec plotproto.ChartSpec) {
	p.Title.Padding = vg.Points(16)
	p.X.Label.Padding = vg.Points(12)
	p.Y.Label.Padding = vg.Points(8)
	if !spec.Legend {
		return
	}
	position := spec.LegendPosition
	if position == "" {
		position = defaultLegendPosition
	}
	p.Legend.Padding = vg.Points(5)
	p.Legend.ThumbnailWidth = vg.Points(24)
	switch position {
	case "topLeft":
		p.Legend.Top, p.Legend.Left = true, true
		p.Legend.XOffs, p.Legend.YOffs = vg.Points(10), vg.Points(-10)
	case "bottomRight":
		p.Legend.Top, p.Legend.Left = false, false
		p.Legend.XOffs, p.Legend.YOffs = vg.Points(-10), vg.Points(10)
	case "bottomLeft":
		p.Legend.Top, p.Legend.Left = false, true
		p.Legend.XOffs, p.Legend.YOffs = vg.Points(10), vg.Points(10)
	default: // topRight is the v2.4 default.
		p.Legend.Top, p.Legend.Left = true, false
		p.Legend.XOffs, p.Legend.YOffs = vg.Points(-10), vg.Points(-10)
	}
}

var sharedMathTextHandler *mathTextHandler

func plotTextHandler() *mathTextHandler {
	if sharedMathTextHandler == nil {
		sharedMathTextHandler = newMathTextHandler()
	}
	return sharedMathTextHandler
}

func init() {
	// Every Plot text site (axes, title, legend, and categorical ticks) reads
	// this one handler from Gonum's existing style defaults.
	plot.DefaultTextHandler = plotTextHandler()
}

// saveWithColorBar writes one chart that carries a colour scale: the chart
// fills the canvas except for the strip the scale occupies.
func saveWithColorBar(p, bar *plot.Plot, path, format string, width, height vg.Length) error {
	canvasWriter, err := draw.NewFormattedCanvas(width, height, format)
	if err != nil {
		return fmt.Errorf("preparing %s canvas: %w", format, err)
	}
	dc := draw.New(canvasWriter)
	marginL := vg.Points(24)
	marginR := vg.Points(24)
	marginB := vg.Points(24)
	marginT := vg.Points(24)
	cropped := draw.Crop(dc, marginL, -marginR, marginB, -marginT)
	main, scale := splitForColorBar(cropped)
	da := p.DataCanvas(main)
	plotTextHandler().titleShiftX = da.Center().X - main.Center().X
	p.Draw(main)
	plotTextHandler().titleShiftX = 0
	bar.Draw(scale)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()
	if _, err := canvasWriter.WriteTo(file); err != nil {
		return fmt.Errorf("writing %s: %w", format, err)
	}
	return nil
}

func savePlot(p *plot.Plot, path, format string, width, height vg.Length) error {
	canvasWriter, err := draw.NewFormattedCanvas(width, height, format)
	if err != nil {
		return fmt.Errorf("preparing %s canvas: %w", format, err)
	}
	dc := draw.New(canvasWriter)
	marginL := vg.Points(24)
	marginR := vg.Points(24)
	marginB := vg.Points(24)
	marginT := vg.Points(24)
	cropped := draw.Crop(dc, marginL, -marginR, marginB, -marginT)
	da := p.DataCanvas(cropped)
	plotTextHandler().titleShiftX = da.Center().X - cropped.Center().X
	p.Draw(cropped)
	plotTextHandler().titleShiftX = 0
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()
	if _, err := canvasWriter.WriteTo(file); err != nil {
		return fmt.Errorf("writing %s: %w", format, err)
	}
	return nil
}

type legendBackground struct {
	width  vg.Length
	height vg.Length
}

func (lb legendBackground) Plot(c draw.Canvas, plt *plot.Plot) {
	if lb.width <= 0 || lb.height <= 0 {
		return
	}
	pad := vg.Points(6)
	var minX, maxX, minY, maxY vg.Length
	if plt.Legend.Left {
		minX = c.Min.X + plt.Legend.XOffs - pad
		maxX = minX + lb.width + pad*2
	} else {
		maxX = c.Max.X + plt.Legend.XOffs + pad
		minX = maxX - lb.width - pad*2
	}
	if plt.Legend.Top {
		maxY = c.Max.Y + plt.Legend.YOffs + pad
		minY = maxY - lb.height - pad*2
	} else {
		minY = c.Min.Y + plt.Legend.YOffs - pad
		maxY = minY + lb.height + pad*2
	}
	pts := []vg.Point{
		{X: minX, Y: minY},
		{X: maxX, Y: minY},
		{X: maxX, Y: maxY},
		{X: minX, Y: maxY},
	}
	c.FillPolygon(color.White, pts)
	c.StrokeLines(draw.LineStyle{
		Color: color.RGBA{R: 215, G: 215, B: 215, A: 255},
		Width: vg.Points(0.5),
	}, append(pts, pts[0]))
}

func addLegendBackground(p *plot.Plot, spec plotproto.ChartSpec) {
	var maxLabelW vg.Length
	count := 0
	for _, s := range spec.Series {
		if s.Label != "" {
			count++
			w := p.Legend.TextStyle.Width(s.Label)
			if w > maxLabelW {
				maxLabelW = w
			}
		}
	}
	if count == 0 {
		return
	}
	totalW := p.Legend.ThumbnailWidth + maxLabelW + vg.Points(22)
	totalH := vg.Points(18)*vg.Length(count) + vg.Points(8)
	p.Add(legendBackground{width: totalW, height: totalH})
}

func toXYs(x, y []float64) plotter.XYs {
	points := make(plotter.XYs, len(x))
	for i := range x {
		points[i].X, points[i].Y = x[i], y[i]
	}
	return points
}

func addSeries(p *plot.Plot, spec plotproto.ChartSpec) error {
	for _, series := range spec.Series {
		if err := validateSeriesStyle(series); err != nil {
			return err
		}
		points := toXYs(series.X, series.Y)
		switch series.Kind {
		case "line":
			line, err := plotter.NewLine(points)
			if err != nil {
				return fmt.Errorf("building line series: %w", err)
			}
			line.LineStyle = seriesLineStyle(series)
			p.Add(line)
			var marker *plotter.Scatter
			if marker = seriesScatter(points, series); marker != nil {
				p.Add(marker)
			}
			if spec.Legend && series.Label != "" {
				if marker != nil {
					p.Legend.Add(series.Label, line, marker)
				} else {
					p.Legend.Add(series.Label, line)
				}
			}
		case "scatter":
			scatter, err := plotter.NewScatter(points)
			if err != nil {
				return fmt.Errorf("building scatter series: %w", err)
			}
			if marker := seriesMarkerStyle(series); marker != nil {
				scatter.GlyphStyle = *marker
			} else {
				// Marker.none is a valid explicit choice; a scatter without a
				// glyph should not silently revert to Gonum's default glyph.
				scatter.GlyphStyle.Shape = nil
			}
			p.Add(scatter)
			if spec.Legend && series.Label != "" {
				p.Legend.Add(series.Label, scatter)
			}
		default:
			return fmt.Errorf("unsupported series kind %q", series.Kind)
		}
	}
	if spec.Legend {
		addLegendBackground(p, spec)
	}
	p.Add(plotFrame{})
	applyAxisPadding(p)
	return nil
}

type plotFrame struct{}

func (plotFrame) Plot(c draw.Canvas, plt *plot.Plot) {
	pts := []vg.Point{
		{X: c.Min.X, Y: c.Max.Y},
		{X: c.Max.X, Y: c.Max.Y},
		{X: c.Max.X, Y: c.Min.Y},
	}
	c.StrokeLines(draw.LineStyle{
		Color: color.RGBA{R: 200, G: 200, B: 200, A: 255},
		Width: vg.Points(0.5),
	}, pts)
}

func applyAxisPadding(p *plot.Plot) {
	if p.X.Max > p.X.Min {
		spanX := p.X.Max - p.X.Min
		padX := spanX * 0.04
		p.X.Min -= padX
		p.X.Max += padX
	}
	if p.Y.Max > p.Y.Min {
		spanY := p.Y.Max - p.Y.Min
		padY := spanY * 0.04
		p.Y.Min -= padY
		p.Y.Max += padY
	}
}

func validateSeriesStyle(series plotproto.SeriesSpec) error {
	if series.Kind != "line" && series.Kind != "scatter" {
		return fmt.Errorf("unsupported series kind %q", series.Kind)
	}
	lineStyle := series.LineStyle
	if lineStyle == "" {
		lineStyle = "solid"
	}
	if lineStyle != "solid" && lineStyle != "dashed" && lineStyle != "dotted" && lineStyle != "dashDot" {
		return fmt.Errorf("unknown line style %q", series.LineStyle)
	}
	if series.Kind != "line" && lineStyle != "solid" {
		return fmt.Errorf("line style applies only to line series")
	}
	if series.LineWidth != 0 && (math.IsNaN(series.LineWidth) || math.IsInf(series.LineWidth, 0) || series.LineWidth <= 0 || series.LineWidth > 32) {
		return fmt.Errorf("line width must be finite and between 0 and 32")
	}
	marker := series.Marker
	if marker == "" {
		if series.Kind == "scatter" {
			marker = "circle"
		} else {
			marker = "none"
		}
	}
	if marker != "none" && marker != "circle" && marker != "square" && marker != "triangle" && marker != "diamond" && marker != "cross" {
		return fmt.Errorf("unknown marker %q", series.Marker)
	}
	if series.MarkerSize != 0 && (math.IsNaN(series.MarkerSize) || math.IsInf(series.MarkerSize, 0) || series.MarkerSize <= 0 || series.MarkerSize > 64) {
		return fmt.Errorf("marker size must be finite and between 0 and 64")
	}
	return nil
}

func seriesLineStyle(series plotproto.SeriesSpec) draw.LineStyle {
	width := series.LineWidth
	if width <= 0 {
		width = 1
	}
	style := draw.LineStyle{Color: plotter.DefaultLineStyle.Color, Width: vg.Points(width)}
	switch series.LineStyle {
	case "dashed":
		style.Dashes = []vg.Length{vg.Points(6), vg.Points(4)}
	case "dotted":
		style.Dashes = []vg.Length{vg.Points(1), vg.Points(3)}
	case "dashDot":
		style.Dashes = []vg.Length{vg.Points(6), vg.Points(3), vg.Points(1), vg.Points(3)}
	}
	return style
}

func seriesScatter(points plotter.XYs, series plotproto.SeriesSpec) *plotter.Scatter {
	return seriesMarkerPlotter(points, series)
}

func seriesMarkerPlotter(points plotter.XYs, series plotproto.SeriesSpec) *plotter.Scatter {
	style := seriesMarkerStyle(series)
	if style == nil {
		return nil
	}
	scatter, err := plotter.NewScatter(points)
	if err != nil {
		return nil
	}
	scatter.GlyphStyle = *style
	return scatter
}

func seriesMarkerStyle(series plotproto.SeriesSpec) *draw.GlyphStyle {
	marker := series.Marker
	if marker == "" {
		if series.Kind == "scatter" {
			marker = "circle"
		} else {
			marker = "none"
		}
	}
	if marker == "none" {
		return nil
	}
	size := series.MarkerSize
	if size <= 0 {
		size = 5
	}
	var shape draw.GlyphDrawer
	switch marker {
	case "circle":
		// RingGlyph is Gonum's v2.3 scatter default: keeping it here makes
		// an unstylised Scatter byte-for-byte comparable in appearance.
		shape = draw.RingGlyph{}
	case "square":
		shape = draw.SquareGlyph{}
	case "triangle":
		shape = draw.TriangleGlyph{}
	case "diamond":
		shape = diamondGlyph{}
	case "cross":
		shape = draw.CrossGlyph{}
	default:
		return nil
	}
	return &draw.GlyphStyle{Color: plotter.DefaultGlyphStyle.Color, Radius: vg.Points(size / 2), Shape: shape}
}

type diamondGlyph struct{}

func (diamondGlyph) DrawGlyph(c *draw.Canvas, style draw.GlyphStyle, point vg.Point) {
	radius := style.Radius
	path := make(vg.Path, 0, 5)
	path.Move(vg.Point{X: point.X, Y: point.Y + radius})
	path.Line(vg.Point{X: point.X + radius, Y: point.Y})
	path.Line(vg.Point{X: point.X, Y: point.Y - radius})
	path.Line(vg.Point{X: point.X - radius, Y: point.Y})
	path.Close()
	c.SetLineStyle(draw.LineStyle{Color: style.Color, Width: vg.Points(0.5)})
	c.Stroke(path)
}

func addBar(p *plot.Plot, spec plotproto.ChartSpec) error {
	bars, err := plotter.NewBarChart(plotter.Values(spec.BarValues), vg.Points(20))
	if err != nil {
		return fmt.Errorf("building bar chart: %w", err)
	}
	p.Add(bars)
	p.NominalX(spec.BarLabels...)
	return nil
}

func addHistogram(p *plot.Plot, spec plotproto.ChartSpec) error {
	histogram, err := plotter.NewHist(plotter.Values(spec.HistogramValues), spec.HistogramBins)
	if err != nil {
		return fmt.Errorf("building histogram: %w", err)
	}
	p.Add(histogram)
	return nil
}

func addBox(p *plot.Plot, spec plotproto.ChartSpec) error {
	box, err := plotter.NewBoxPlot(vg.Points(40), 0, plotter.Values(spec.BoxValues))
	if err != nil {
		return fmt.Errorf("building box plot: %w", err)
	}
	p.Add(box)
	p.NominalX("")
	return nil
}

// ahdErrorBarData adapts parallel x/y/lowerError/upperError slices to the
// XYer + YErrorer interfaces plotter.NewYErrorBars requires.
type ahdErrorBarData struct {
	x, y, lower, upper []float64
}

func (d ahdErrorBarData) Len() int                        { return len(d.x) }
func (d ahdErrorBarData) XY(i int) (float64, float64)     { return d.x[i], d.y[i] }
func (d ahdErrorBarData) YError(i int) (float64, float64) { return d.lower[i], d.upper[i] }

func addErrorBar(p *plot.Plot, spec plotproto.ChartSpec) error {
	data := ahdErrorBarData{x: spec.ErrorX, y: spec.ErrorY, lower: spec.ErrorLower, upper: spec.ErrorUpper}
	scatter, err := plotter.NewScatter(toXYs(spec.ErrorX, spec.ErrorY))
	if err != nil {
		return fmt.Errorf("building error bar points: %w", err)
	}
	p.Add(scatter)
	bars, err := plotter.NewYErrorBars(data)
	if err != nil {
		return fmt.Errorf("building error bars: %w", err)
	}
	p.Add(bars)
	return nil
}
