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
		p, err := buildPlot(chart)
		if err != nil {
			return err
		}
		if err := p.Save(width, height, request.OutputPath); err != nil {
			return fmt.Errorf("saving %s: %w", format, err)
		}
		return nil
	}

	return renderGrid(request, format, width, height)
}

// renderGrid composes a rows x columns Figure. An absent cell (fewer charts
// than grid cells) is left blank: its Plot has no title, labels, or
// plotters, matching the documented "blank remaining cells" subplot policy.
func renderGrid(request plotproto.Request, format string, width, height vg.Length) error {
	plots := make([][]*plot.Plot, request.Rows)
	for row := 0; row < request.Rows; row++ {
		plots[row] = make([]*plot.Plot, request.Columns)
		for column := 0; column < request.Columns; column++ {
			spec := request.Charts[row*request.Columns+column]
			if !spec.Present {
				plots[row][column] = plot.New()
				plots[row][column].HideAxes()
				continue
			}
			p, err := buildPlot(spec)
			if err != nil {
				return err
			}
			plots[row][column] = p
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
			plots[row][column].Draw(aligned[row][column])
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

func buildPlot(spec plotproto.ChartSpec) (*plot.Plot, error) {
	p := plot.New()
	p.Title.Text = spec.Title
	p.X.Label.Text = spec.XLabel
	p.Y.Label.Text = spec.YLabel

	switch spec.Kind {
	case "empty":
		// A blank canvas with only its metadata: Plot.new() with no series
		// added yet.
	case "line-scatter":
		if err := addSeries(p, spec); err != nil {
			return nil, err
		}
	case "bar":
		if err := addBar(p, spec); err != nil {
			return nil, err
		}
	case "histogram":
		if err := addHistogram(p, spec); err != nil {
			return nil, err
		}
	case "box":
		if err := addBox(p, spec); err != nil {
			return nil, err
		}
	case "errorBar":
		if err := addErrorBar(p, spec); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported chart kind %q", spec.Kind)
	}
	return p, nil
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
		points := toXYs(series.X, series.Y)
		switch series.Kind {
		case "line":
			line, err := plotter.NewLine(points)
			if err != nil {
				return fmt.Errorf("building line series: %w", err)
			}
			p.Add(line)
			if spec.Legend && series.Label != "" {
				p.Legend.Add(series.Label, line)
			}
		case "scatter":
			scatter, err := plotter.NewScatter(points)
			if err != nil {
				return fmt.Errorf("building scatter series: %w", err)
			}
			p.Add(scatter)
			if spec.Legend && series.Label != "" {
				p.Legend.Add(series.Label, scatter)
			}
		default:
			return fmt.Errorf("unsupported series kind %q", series.Kind)
		}
	}
	return nil
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
