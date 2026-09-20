package main

// The pie chart (v2.2). Gonum has no pie plotter, so this is one small
// plot.Plotter drawn with the same vg primitives every other Gonum plotter
// uses -- no second rendering engine and no extra dependency.
//
// A pie has no Cartesian axes, so the Plot it lives on hides them and the
// plotter simply fills the canvas: wedges clockwise from twelve o'clock, in
// the order the categories were given, from one fixed palette.

import (
	"fmt"
	"image/color"
	"math"
	"unicode/utf8"

	"ahdcode/internal/plotproto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// piePalette is AhdCode's one categorical palette. It is fixed in v2.2:
// there is no palette argument, so a chart drawn twice is drawn the same.
// The twelve hues are distinguishable in order and stay legible behind black
// percentage text; a pie with more slices cycles through them again, which
// is why a legend is the pie's default key.
var piePalette = []color.Color{
	color.RGBA{R: 0x4C, G: 0x78, B: 0xA8, A: 0xFF}, // blue
	color.RGBA{R: 0xF5, G: 0x85, B: 0x18, A: 0xFF}, // orange
	color.RGBA{R: 0x54, G: 0xA2, B: 0x4B, A: 0xFF}, // green
	color.RGBA{R: 0xE4, G: 0x55, B: 0x56, A: 0xFF}, // red
	color.RGBA{R: 0x79, G: 0x70, B: 0xA3, A: 0xFF}, // purple
	color.RGBA{R: 0x8C, G: 0x61, B: 0x3C, A: 0xFF}, // brown
	color.RGBA{R: 0xD6, G: 0x77, B: 0xAD, A: 0xFF}, // pink
	color.RGBA{R: 0x7F, G: 0x7F, B: 0x7F, A: 0xFF}, // grey
	color.RGBA{R: 0xBA, G: 0xB0, B: 0x3C, A: 0xFF}, // olive
	color.RGBA{R: 0x4B, G: 0xB2, B: 0xC4, A: 0xFF}, // teal
	color.RGBA{R: 0xA8, G: 0x78, B: 0x4C, A: 0xFF}, // tan
	color.RGBA{R: 0x94, G: 0x4C, B: 0xA8, A: 0xFF}, // violet
}

// pieChart draws one pie. It holds only what it draws; the Chart's title and
// legend belong to the plot.Plot around it.
type pieChart struct {
	labels []string
	values []float64
	total  float64
	// withLegend leaves room for the key Gonum draws over the same canvas.
	withLegend bool
}

func newPieChart(labels []string, values []float64) (*pieChart, error) {
	total := 0.0
	for _, value := range values {
		total += value
	}
	if total <= 0 {
		return nil, fmt.Errorf("a pie chart needs at least one value greater than zero")
	}
	return &pieChart{labels: labels, values: values, total: total}, nil
}

// sliceColor is the palette entry one slice uses, cycling for a pie with
// more slices than colours.
func sliceColor(index int) color.Color {
	return piePalette[index%len(piePalette)]
}

// DataRange keeps the plot's own scaling out of the way; the pie is drawn in
// canvas coordinates, not data coordinates.
func (pie *pieChart) DataRange() (xmin, xmax, ymin, ymax float64) { return 0, 1, 0, 1 }

// pieLabelThreshold is the smallest share that gets its percentage written
// on the slice. Below it the text would not fit inside the wedge, and the
// legend already names the category. It is a constant, so the same data
// always produces the same picture.
const pieLabelThreshold = 0.05

func (pie *pieChart) Plot(canvas draw.Canvas, _ *plot.Plot) {
	rectangle := canvas.Rectangle
	width, height := rectangle.Size().X, rectangle.Size().Y
	radius := width
	if height < radius {
		radius = height
	}
	share := 0.86
	if pie.withLegend {
		// The key is drawn over this same canvas, so the circle steps back
		// from it rather than sliding underneath.
		share = 0.74
	}
	radius = radius / 2 * vg.Length(share)
	if radius <= 0 {
		return
	}
	centre := vg.Point{
		X: rectangle.Min.X + rectangle.Size().X/2,
		Y: rectangle.Min.Y + rectangle.Size().Y/2,
	}

	// Wedges run clockwise from twelve o'clock, which is how a reader
	// expects the first category to appear.
	start := math.Pi / 2
	edge := draw.LineStyle{Color: color.White, Width: vg.Points(1)}
	text := draw.TextStyle{
		Color:  color.Black,
		Font:   plot.DefaultFont,
		XAlign: draw.XCenter,
		YAlign: draw.YCenter,
	}
	text.Font.Size = vg.Points(11)
	text.Handler = plot.DefaultTextHandler

	for index, value := range pie.values {
		share := value / pie.total
		if share <= 0 {
			// A zero slice is valid data and simply has no wedge; the
			// legend still lists the category.
			continue
		}
		sweep := -2 * math.Pi * share
		var wedge vg.Path
		wedge.Move(centre)
		wedge.Line(vg.Point{
			X: centre.X + radius*vg.Length(math.Cos(start)),
			Y: centre.Y + radius*vg.Length(math.Sin(start)),
		})
		wedge.Arc(centre, radius, start, sweep)
		wedge.Close()
		canvas.SetColor(sliceColor(index))
		canvas.Fill(wedge)
		canvas.StrokeLines(edge, wedgeOutline(centre, radius, start, sweep))

		if share >= pieLabelThreshold {
			middle := start + sweep/2
			at := vg.Point{
				X: centre.X + radius*0.62*vg.Length(math.Cos(middle)),
				Y: centre.Y + radius*0.62*vg.Length(math.Sin(middle)),
			}
			canvas.FillText(text, at, fmt.Sprintf("%.0f%%", share*100))
		}
		start += sweep
	}
}

// wedgeOutline is the wedge's boundary as line segments, so the white
// separator between neighbouring slices is stroked the same way in every
// output format.
func wedgeOutline(centre vg.Point, radius vg.Length, start, sweep float64) []vg.Point {
	const steps = 64
	count := int(math.Ceil(math.Abs(sweep) / (2 * math.Pi) * steps))
	if count < 2 {
		count = 2
	}
	points := make([]vg.Point, 0, count+3)
	points = append(points, centre)
	for step := 0; step <= count; step++ {
		angle := start + sweep*float64(step)/float64(count)
		points = append(points, vg.Point{
			X: centre.X + radius*vg.Length(math.Cos(angle)),
			Y: centre.Y + radius*vg.Length(math.Sin(angle)),
		})
	}
	points = append(points, centre)
	return points
}

// pieSwatch is a legend key: one filled square in a slice's colour.
type pieSwatch struct{ fill color.Color }

func (swatch pieSwatch) Thumbnail(canvas *draw.Canvas) {
	canvas.SetColor(swatch.fill)
	var box vg.Path
	box.Move(vg.Point{X: canvas.Min.X, Y: canvas.Min.Y})
	box.Line(vg.Point{X: canvas.Max.X, Y: canvas.Min.Y})
	box.Line(vg.Point{X: canvas.Max.X, Y: canvas.Max.Y})
	box.Line(vg.Point{X: canvas.Min.X, Y: canvas.Max.Y})
	box.Close()
	canvas.Fill(box)
}

// validatePie checks one pie spec exactly as the runtime already did. The
// helper is a separate executable reading JSON from a file, so it repeats
// every rule rather than trusting its input.
func validatePie(spec plotproto.ChartSpec) error {
	if len(spec.PieLabels) == 0 {
		return fmt.Errorf("a pie chart needs at least one slice")
	}
	if len(spec.PieLabels) != len(spec.PieValues) {
		return fmt.Errorf("a pie chart has %d labels and %d values", len(spec.PieLabels), len(spec.PieValues))
	}
	if len(spec.PieLabels) > plotproto.MaxPieSlices {
		return fmt.Errorf("a pie chart draws at most %d slices; got %d", plotproto.MaxPieSlices, len(spec.PieLabels))
	}
	for _, label := range spec.PieLabels {
		if err := validateLabel(label); err != nil {
			return err
		}
	}
	positive := false
	for _, value := range spec.PieValues {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("pie values must be finite")
		}
		if value < 0 {
			return fmt.Errorf("pie values must be non-negative")
		}
		if value > 0 {
			positive = true
		}
	}
	if !positive {
		return fmt.Errorf("a pie chart needs at least one value greater than zero")
	}
	// A pie has no axes, so axis titles cannot be carried on one.
	if spec.XLabel != "" || spec.YLabel != "" {
		return fmt.Errorf("a pie chart has no x or y axis")
	}
	return nil
}

func validateLabel(label string) error {
	if !utf8.ValidString(label) {
		return fmt.Errorf("a category label must be valid UTF-8")
	}
	if utf8.RuneCountInString(label) > plotproto.MaxLabelRunes {
		return fmt.Errorf("a category label is at most %d characters", plotproto.MaxLabelRunes)
	}
	return nil
}

func addPie(p *plot.Plot, spec plotproto.ChartSpec) error {
	if err := validatePie(spec); err != nil {
		return err
	}
	pie, err := newPieChart(spec.PieLabels, spec.PieValues)
	if err != nil {
		return err
	}
	pie.withLegend = spec.Legend
	// A pie is not plotted against axes, and Gonum would otherwise draw the
	// 0..1 range DataRange reports.
	p.HideAxes()
	p.Add(pie)
	if spec.Legend {
		for index, label := range spec.PieLabels {
			p.Legend.Add(label, pieSwatch{fill: sliceColor(index)})
		}
		p.Legend.Top = true
		p.Legend.Left = false
	}
	return nil
}
