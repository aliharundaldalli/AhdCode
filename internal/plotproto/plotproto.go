// Package plotproto is the narrow, stable request/response protocol between
// AhdCode's Plot standard module and the bundled ahdplot renderer helper.
//
// Gonum's plotting library cannot be linked into ahdruntime.go: that file is
// embedded verbatim into every natively-compiled AhdCode program, which
// builds in an isolated, dependency-free workspace with no vendoring or
// network access (see internal/build/pipeline.go). Chart rendering therefore
// happens out-of-process, in a small bundled ahdplot executable that does
// depend on Gonum. Both the persistent evaluator (internal/evaluator/plot.go)
// and the native runtime (ahdruntime.go, which duplicates this protocol's
// shape locally since it cannot import this package) drive that helper the
// same way: write a Request as JSON to a temporary file, run
// `ahdplot <request-file>`, and read a Response as JSON from its stdout.
package plotproto

// Request describes one rendering job: a rows x columns grid of chart specs
// (row-major, length rows*columns), rendered together into one image at
// OutputPath. A single Chart.save/Chart.show is the degenerate 1x1 case.
type Request struct {
	Mode       string      `json:"mode,omitempty"` // empty for charts, "math" for one math-text image
	OutputPath string      `json:"output_path"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	Rows       int         `json:"rows"`
	Columns    int         `json:"columns"`
	Charts     []ChartSpec `json:"charts"`
	Math       *MathSpec   `json:"math,omitempty"`
}

// MathSpec is the narrow helper protocol used by the Surface viewer for one
// cached math label. It is intentionally an image request so the viewer does
// not need Gonum or another plotting dependency.
type MathSpec struct {
	Formula  string  `json:"formula"`
	FontSize float64 `json:"font_size"`
}

// ChartSpec is the rendering-relevant content of one Chart. Present is false
// for an empty subplot cell (fewer charts than grid cells), which is left
// blank rather than rendered.
type ChartSpec struct {
	Present bool   `json:"present"`
	Kind    string `json:"kind"` // "line-scatter" | "bar" | "histogram" | "box" | "errorBar" | "pie" | "heatmap" | "empty"

	Title  string `json:"title,omitempty"`
	XLabel string `json:"x_label,omitempty"`
	YLabel string `json:"y_label,omitempty"`
	Legend bool   `json:"legend,omitempty"`
	// LegendPosition is one of topRight, topLeft, bottomRight, or
	// bottomLeft. Empty keeps the renderer's sensible topRight default for
	// requests produced by older clients.
	LegendPosition string `json:"legend_position,omitempty"`

	Series []SeriesSpec `json:"series,omitempty"` // kind == "line-scatter"

	BarLabels []string  `json:"bar_labels,omitempty"` // kind == "bar"
	BarValues []float64 `json:"bar_values,omitempty"`

	HistogramValues []float64 `json:"histogram_values,omitempty"` // kind == "histogram"
	HistogramBins   int       `json:"histogram_bins,omitempty"`

	BoxValues []float64 `json:"box_values,omitempty"` // kind == "box"

	ErrorX     []float64 `json:"error_x,omitempty"` // kind == "errorBar"
	ErrorY     []float64 `json:"error_y,omitempty"`
	ErrorLower []float64 `json:"error_lower,omitempty"`
	ErrorUpper []float64 `json:"error_upper,omitempty"`

	// kind == "pie" (v2.2). One slice per label, in the order given; Legend
	// shows the category key.
	PieLabels []string  `json:"pie_labels,omitempty"`
	PieValues []float64 `json:"pie_values,omitempty"`

	// kind == "heatmap" (v2.2). HeatmapValues is row-major: one row per y
	// label and one column per x label, so values[row][column] is the cell
	// at yLabels[row] and xLabels[column]. Legend shows the colour scale.
	HeatmapXLabels []string    `json:"heatmap_x_labels,omitempty"`
	HeatmapYLabels []string    `json:"heatmap_y_labels,omitempty"`
	HeatmapValues  [][]float64 `json:"heatmap_values,omitempty"`
}

// The bounds ahdplot enforces on a request, whatever produced it. AhdCode
// itself checks the same rules before sending, but the helper is a separate
// executable that reads JSON from a file and trusts none of it.
const (
	// MaxPieSlices keeps a pie readable and its category palette distinct.
	MaxPieSlices = 64
	// MaxHeatmapLabels and MaxHeatmapCells match the Surface grid bound: a
	// 256 x 256 grid is already far past what a labelled chart can show.
	MaxHeatmapLabels = 256
	MaxHeatmapCells  = 65536
	// MaxLabelRunes bounds one category label.
	MaxLabelRunes = 200
)

// SeriesSpec is one line or scatter series on a line-scatter Chart.
type SeriesSpec struct {
	Kind       string    `json:"kind"` // "line" | "scatter"
	Label      string    `json:"label,omitempty"`
	X          []float64 `json:"x"`
	Y          []float64 `json:"y"`
	LineStyle  string    `json:"line_style,omitempty"`
	LineWidth  float64   `json:"line_width,omitempty"`
	Marker     string    `json:"marker,omitempty"`
	MarkerSize float64   `json:"marker_size,omitempty"`
}

// Response is ahdplot's sole stdout output: one line of JSON.
type Response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}
