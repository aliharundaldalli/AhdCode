package main

// The heatmap (v2.2). The colour map and the colour bar are Gonum's --
// palette/moreland supplies a sequential, luminance-ordered scale and
// plotter.ColorBar draws it -- but the grid itself is one small
// plot.Plotter here rather than plotter.HeatMap.
//
// Gonum's HeatMap centres each cell on its grid point, so half of the
// outermost cells falls outside the axes, and it skips any cell whose far
// corner is not inside the drawing area. A category is a cell, not a point,
// so this plotter instead puts cell (column, row) between the integers
// column..column+1 and row..row+1 and fills it: every cell is whole, every
// tick sits at a cell's centre, and the grid meets the axes exactly.
//
// Drawing the cells here is also what lets each one be outlined, which the
// scale needs -- see heatmapEdge.
//
// The colour bar is its own small plot drawn into a strip of the chart's
// canvas, because Gonum's ColorBar is a plotter that fills whatever plot it
// belongs to. That keeps one heatmap one cell, so a heatmap works as a
// Figure subplot exactly like every other chart.

import (
	"fmt"
	"image/color"
	"math"

	"ahdcode/internal/plotproto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// heatmapColorMap is AhdCode's one heatmap colour map: Moreland's black
// body, a sequential scale whose luminance rises monotonically from dark to
// light, so it reads correctly in greyscale and for the common colour-vision
// deficiencies, and low and high are never confusable. There is no colormap
// argument in v2.2.
//
// Its top colour is pure white, which would vanish against the page, so the
// scale is shortened: the data spans the darkest colour up to a bright
// yellow. The colour bar is built from the same shortened map, so a cell and
// the scale beside it always agree.
func heatmapColorMap() palette.ColorMap {
	return &shortenedColorMap{full: moreland.BlackBody(), top: heatmapTopShare}
}

// heatmapTopShare is how much of the underlying scale the data uses. The
// last few percent are the near-white end.
const heatmapTopShare = 0.88

// shortenedColorMap presents a colour map that uses only the first `top`
// share of another one. It is the whole ColorMap interface because both the
// cells and Gonum's ColorBar ask for colours through it.
type shortenedColorMap struct {
	full     palette.ColorMap
	top      float64
	min, max float64
}

func (m *shortenedColorMap) At(value float64) (color.Color, error) {
	if value < m.min || value > m.max || m.max <= m.min {
		return nil, palette.ErrOverflow
	}
	share := (value - m.min) / (m.max - m.min) * m.top
	return m.full.At(m.full.Min() + share*(m.full.Max()-m.full.Min()))
}

func (m *shortenedColorMap) Max() float64       { return m.max }
func (m *shortenedColorMap) Min() float64       { return m.min }
func (m *shortenedColorMap) Alpha() float64     { return m.full.Alpha() }
func (m *shortenedColorMap) SetAlpha(a float64) { m.full.SetAlpha(a) }

func (m *shortenedColorMap) SetMax(value float64) {
	m.max = value
	m.full.SetMax(value)
	m.full.SetMin(m.min)
}

func (m *shortenedColorMap) SetMin(value float64) {
	m.min = value
	m.full.SetMin(value)
	m.full.SetMax(m.max)
}

func (m *shortenedColorMap) Palette(colors int) palette.Palette {
	if colors < 2 {
		colors = 2
	}
	shades := make([]color.Color, colors)
	for index := range shades {
		value := m.min + (m.max-m.min)*float64(index)/float64(colors-1)
		shade, err := m.At(value)
		if err != nil {
			shade = color.White
		}
		shades[index] = shade
	}
	return fixedPalette(shades)
}

// fixedPalette is a palette of already-chosen colours.
type fixedPalette []color.Color

func (p fixedPalette) Colors() []color.Color { return p }

// heatmapColors is how finely the sequential scale is sampled. Cells are
// coloured by index into that sample rather than by asking the colour map
// for an exact value: a value sitting precisely on the range's end is an
// ordinary cell, not an out-of-range error.
const heatmapColors = 255

// heatmapGrid draws the cells. values is row-major: one row per y label and
// one column per x label.
type heatmapGrid struct {
	values    [][]float64
	colors    []color.Color
	low, high float64
}

func (g *heatmapGrid) columns() int {
	if len(g.values) == 0 {
		return 0
	}
	return len(g.values[0])
}

func (g *heatmapGrid) rows() int { return len(g.values) }

// DataRange makes one cell one unit wide and one unit tall, so cell
// boundaries fall on integers and the whole grid fits the axes exactly.
func (g *heatmapGrid) DataRange() (xmin, xmax, ymin, ymax float64) {
	return 0, float64(g.columns()), 0, float64(g.rows())
}

// heatmapEdge outlines each cell. The scale's highest colour is white, so
// without an outline the warmest cell would be invisible against the page;
// the outline also keeps two equal neighbours from reading as one block.
var heatmapEdge = draw.LineStyle{Color: color.Gray{Y: 0x99}, Width: vg.Points(0.5)}

func (g *heatmapGrid) Plot(canvas draw.Canvas, p *plot.Plot) {
	trX, trY := p.Transforms(&canvas)
	for row := range g.values {
		for column, value := range g.values[row] {
			left, right := trX(float64(column)), trX(float64(column+1))
			bottom, top := trY(float64(row)), trY(float64(row+1))
			corners := []vg.Point{
				{X: left, Y: bottom}, {X: right, Y: bottom},
				{X: right, Y: top}, {X: left, Y: top}, {X: left, Y: bottom},
			}
			var cell vg.Path
			cell.Move(corners[0])
			for _, corner := range corners[1:] {
				cell.Line(corner)
			}
			cell.Close()
			canvas.SetColor(g.colorOf(value))
			canvas.Fill(cell)
			canvas.StrokeLines(heatmapEdge, corners)
		}
	}
}

// colorOf places one cell on the sampled scale. The lowest value takes the
// first colour and the highest the last, so every cell always has one.
func (g *heatmapGrid) colorOf(value float64) color.Color {
	share := (value - g.low) / (g.high - g.low)
	index := int(share*float64(len(g.colors)-1) + 0.5)
	if index < 0 {
		index = 0
	}
	if index >= len(g.colors) {
		index = len(g.colors) - 1
	}
	return g.colors[index]
}

// heatmapRange is the lowest and highest cell, which the colour map and the
// colour bar are both scaled to. An all-equal matrix is valid data: its
// range is widened by a whisker so the colour map has something to map.
func heatmapRange(values [][]float64) (low, high float64) {
	low, high = math.Inf(1), math.Inf(-1)
	for _, row := range values {
		for _, value := range row {
			if value < low {
				low = value
			}
			if value > high {
				high = value
			}
		}
	}
	if low == high {
		low, high = low-0.5, high+0.5
	}
	return low, high
}

// validateHeatmap checks one heatmap spec exactly as the runtime already
// did; the helper trusts no input.
func validateHeatmap(spec plotproto.ChartSpec) error {
	columns, rows := len(spec.HeatmapXLabels), len(spec.HeatmapYLabels)
	if columns == 0 || rows == 0 {
		return fmt.Errorf("a heatmap needs at least one x label and one y label")
	}
	if columns > plotproto.MaxHeatmapLabels || rows > plotproto.MaxHeatmapLabels {
		return fmt.Errorf("a heatmap has at most %d labels on each axis; got %d and %d",
			plotproto.MaxHeatmapLabels, columns, rows)
	}
	if columns*rows > plotproto.MaxHeatmapCells {
		return fmt.Errorf("a heatmap draws at most %d cells; got %d", plotproto.MaxHeatmapCells, columns*rows)
	}
	for _, label := range append(append([]string{}, spec.HeatmapXLabels...), spec.HeatmapYLabels...) {
		if err := validateLabel(label); err != nil {
			return err
		}
	}
	if len(spec.HeatmapValues) != rows {
		return fmt.Errorf("the heatmap Matrix has %d rows; one row per y label (%d) is needed",
			len(spec.HeatmapValues), rows)
	}
	for _, row := range spec.HeatmapValues {
		if len(row) != columns {
			return fmt.Errorf("the heatmap Matrix has %d columns; one column per x label (%d) is needed",
				len(row), columns)
		}
		for _, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("heatmap values must be finite; NaN and Infinity cannot be drawn")
			}
		}
	}
	return nil
}

// categoryTicks puts one tick in the middle of each cell, in input order.
func categoryTicks(labels []string) plot.ConstantTicks {
	ticks := make([]plot.Tick, len(labels))
	for index, label := range labels {
		ticks[index] = plot.Tick{Value: float64(index) + 0.5, Label: label}
	}
	return plot.ConstantTicks(ticks)
}

// addHeatmap fills the chart's own plot with the grid. The colour bar, when
// the legend is on, is returned as a second plot for the caller to draw
// beside it.
func addHeatmap(p *plot.Plot, spec plotproto.ChartSpec) (*plot.Plot, error) {
	if err := validateHeatmap(spec); err != nil {
		return nil, err
	}
	low, high := heatmapRange(spec.HeatmapValues)
	colorMap := heatmapColorMap()
	colorMap.SetMin(low)
	colorMap.SetMax(high)

	p.Add(&heatmapGrid{
		values: spec.HeatmapValues,
		colors: colorMap.Palette(heatmapColors).Colors(),
		low:    low, high: high,
	})
	p.X.Tick.Marker = categoryTicks(spec.HeatmapXLabels)
	p.Y.Tick.Marker = categoryTicks(spec.HeatmapYLabels)
	// The grid fills the axes exactly, so no padding is wanted at the ends.
	p.X.Padding, p.Y.Padding = 0, 0

	if !spec.Legend {
		return nil, nil
	}
	bar := plot.New()
	bar.Add(&plotter.ColorBar{ColorMap: colorMap, Vertical: true})
	bar.HideX()
	bar.Y.Padding = 0
	return bar, nil
}

// heatmapBarFraction is the share of a heatmap's canvas the colour scale
// takes. It is a constant, so a heatmap is laid out the same every time.
const heatmapBarFraction = 0.14

// splitForColorBar divides one chart canvas into the grid's area and the
// colour scale's strip beside it.
func splitForColorBar(canvas draw.Canvas) (main, bar draw.Canvas) {
	barWidth := (canvas.Max.X - canvas.Min.X) * heatmapBarFraction
	main, bar = canvas, canvas
	main.Max.X = canvas.Max.X - barWidth
	bar.Min.X = canvas.Max.X - barWidth
	return main, bar
}
