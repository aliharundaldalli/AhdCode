package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strconv"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// A Surface is drawn by this helper with a small, deterministic software
// projection: the grid is scaled into a box, turned by the camera, projected
// orthographically, and painted back to front as shaded quadrilaterals
// (the painter's algorithm, which is exact for a height field seen from one
// side). There is no 3D engine, no GPU state, no lighting system, and no
// scene: one fixed light shades the surface for depth, and one fixed
// height-based color scale colors it. The same code renders the interactive
// view and a saved PNG, so Surface.save is byte-for-byte reproducible on one
// machine.

// Surface bounds: each axis holds 2 to maxGrid values, and the whole grid at
// most maxVertices points.
const (
	maxGrid        = 256
	maxVertices    = maxGrid * maxGrid
	maxSurfaceSide = 2000 // size() units on each side
	pixelsPerUnit  = 4.0 / 3.0
)

// surfaceSpec is the Surface as the program sends it.
type surfaceSpec struct {
	X []float64   `json:"x"`
	Y []float64   `json:"y"`
	Z [][]float64 `json:"z"`
	// XCategories and YCategories are presentation labels (v2.2): one per x
	// or y coordinate, shown instead of that coordinate's number. Empty
	// means the axis shows its numbers, exactly as before v2.2.
	XCategories []string `json:"x_categories,omitempty"`
	YCategories []string `json:"y_categories,omitempty"`
	Title       string   `json:"title"`
	XLabel      string   `json:"x_label"`
	YLabel      string   `json:"y_label"`
	ZLabel      string   `json:"z_label"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Wireframe   bool     `json:"wireframe"`
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func increasing(values []float64) bool {
	for index := range values {
		if !finite(values[index]) || (index > 0 && values[index] <= values[index-1]) {
			return false
		}
	}
	return true
}

// validate checks everything the runtime already checked; the helper
// trusts no input.
func (s surfaceSpec) validate() error {
	if len(s.X) < 2 || len(s.Y) < 2 || len(s.X) > maxGrid || len(s.Y) > maxGrid || len(s.X)*len(s.Y) > maxVertices {
		return fmt.Errorf("a Surface needs 2 to %d x and y values", maxGrid)
	}
	if !increasing(s.X) || !increasing(s.Y) {
		return errors.New("Surface x and y values must be finite and increasing")
	}
	if len(s.Z) != len(s.Y) {
		return errors.New("the z Matrix needs one row per y value")
	}
	for _, row := range s.Z {
		if len(row) != len(s.X) {
			return errors.New("the z Matrix needs one column per x value")
		}
		for _, value := range row {
			if !finite(value) {
				return errors.New("Surface z values must be finite")
			}
		}
	}
	if s.Width < 1 || s.Height < 1 || s.Width > maxSurfaceSide || s.Height > maxSurfaceSide {
		return fmt.Errorf("a Surface is 1 to %d units on each side", maxSurfaceSide)
	}
	for _, text := range []string{s.Title, s.XLabel, s.YLabel, s.ZLabel} {
		if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxTitleRunes {
			return errors.New("invalid Surface text")
		}
	}
	// Categories are optional, but when present there is exactly one per
	// coordinate; anything else would mislabel the axis.
	for _, categories := range []struct {
		labels      []string
		coordinates int
		axis        string
	}{{s.XCategories, len(s.X), "x"}, {s.YCategories, len(s.Y), "y"}} {
		if len(categories.labels) == 0 {
			continue
		}
		if len(categories.labels) != categories.coordinates {
			return fmt.Errorf("a Surface needs one %s category per %s value", categories.axis, categories.axis)
		}
		for _, text := range categories.labels {
			if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxTitleRunes {
				return errors.New("invalid Surface category")
			}
		}
	}
	return nil
}

// camera is the viewer's whole state for a Surface: orbit angles in
// degrees, zoom, and pan in output pixels. It belongs to the view, never to
// the Surface.
type camera struct {
	azimuth   float64
	elevation float64
	zoom      float64
	panX      float64
	panY      float64
}

// Camera bounds.
const (
	initialAzimuth   = -55.0
	initialElevation = 28.0
	minElevation     = -85.0
	maxElevation     = 85.0
	minSurfaceZoom   = 0.3
	maxSurfaceZoom   = 6.0
)

func initialCamera() camera {
	return camera{azimuth: initialAzimuth, elevation: initialElevation, zoom: 1}
}

// orbit turns the camera by degrees; the elevation stops short of the poles,
// so the view never flips, and the azimuth wraps.
func (c *camera) orbit(dAzimuth, dElevation float64) {
	if !finite(dAzimuth) || !finite(dElevation) {
		return
	}
	c.azimuth = math.Mod(c.azimuth+dAzimuth+540, 360) - 180
	c.elevation = math.Max(minElevation, math.Min(maxElevation, c.elevation+dElevation))
}

func (c *camera) zoomBy(factor float64) {
	if !(factor > 0) || !finite(factor) {
		return
	}
	c.zoom = math.Max(minSurfaceZoom, math.Min(maxSurfaceZoom, c.zoom*factor))
}

// panBy moves the picture, kept within one picture size of the center so it
// can never be lost.
func (c *camera) panBy(dx, dy, width, height float64) {
	if !finite(dx) || !finite(dy) {
		return
	}
	c.panX = math.Max(-width, math.Min(width, c.panX+dx))
	c.panY = math.Max(-height, math.Min(height, c.panY+dy))
}

// point is one grid point scaled into the box: x and y in -1..1, z in
// -zHalf..zHalf.
type point struct{ x, y, z float64 }

const zHalf = 0.65

// normalized scales the grid into the box. A flat surface is drawn as the
// plane z = 0.
func (s surfaceSpec) normalized() ([][]point, float64, float64) {
	lowZ, highZ := math.Inf(1), math.Inf(-1)
	for _, row := range s.Z {
		for _, value := range row {
			lowZ, highZ = math.Min(lowZ, value), math.Max(highZ, value)
		}
	}
	scale := func(value, low, high, half float64) float64 {
		if high == low {
			return 0
		}
		return (2*(value-low)/(high-low) - 1) * half
	}
	grid := make([][]point, len(s.Y))
	for j, yValue := range s.Y {
		grid[j] = make([]point, len(s.X))
		for i, xValue := range s.X {
			grid[j][i] = point{
				x: scale(xValue, s.X[0], s.X[len(s.X)-1], 1),
				y: scale(yValue, s.Y[0], s.Y[len(s.Y)-1], 1),
				z: scale(s.Z[j][i], lowZ, highZ, zHalf),
			}
		}
	}
	return grid, lowZ, highZ
}

// projection turns box coordinates into image pixels, with a depth that
// grows toward the viewer.
type projection struct {
	right, up, toward [3]float64
	scale             float64
	centerX, centerY  float64
}

func newProjection(c camera, width, height float64, top float64) projection {
	azimuth, elevation := c.azimuth*math.Pi/180, c.elevation*math.Pi/180
	p := projection{
		right:  [3]float64{-math.Sin(azimuth), math.Cos(azimuth), 0},
		up:     [3]float64{-math.Sin(elevation) * math.Cos(azimuth), -math.Sin(elevation) * math.Sin(azimuth), math.Cos(elevation)},
		toward: [3]float64{math.Cos(elevation) * math.Cos(azimuth), math.Cos(elevation) * math.Sin(azimuth), math.Sin(elevation)},
	}
	// The box's half diagonal fits the plot area at zoom 1 whatever the
	// angle, so orbiting never changes the size.
	radius := math.Sqrt(2 + zHalf*zHalf)
	plotHeight := height - top
	p.scale = 0.46 * math.Min(width, plotHeight) / radius * c.zoom
	p.centerX, p.centerY = width/2+c.panX, top+plotHeight/2+c.panY
	return p
}

func dot(a [3]float64, b point) float64 { return a[0]*b.x + a[1]*b.y + a[2]*b.z }

func (p projection) at(q point) (float64, float64, float64) {
	return p.centerX + dot(p.right, q)*p.scale, p.centerY - dot(p.up, q)*p.scale, dot(p.toward, q)
}

// heightColor is the fixed color scale: deep blue through teal and green to
// yellow, by height 0..1.
func heightColor(t float64) [3]float64 {
	stops := [][3]float64{{0.20, 0.18, 0.55}, {0.13, 0.45, 0.60}, {0.15, 0.65, 0.50}, {0.55, 0.80, 0.30}, {0.98, 0.88, 0.25}}
	t = math.Max(0, math.Min(1, t)) * float64(len(stops)-1)
	index := min(int(t), len(stops)-2)
	f := t - float64(index)
	a, b := stops[index], stops[index+1]
	return [3]float64{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f, a[2] + (b[2]-a[2])*f}
}

func toColor(rgb [3]float64, shade float64) color.RGBA {
	channel := func(value float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, value*shade)) * 255)) }
	return color.RGBA{channel(rgb[0]), channel(rgb[1]), channel(rgb[2]), 255}
}

// quad is one grid cell, projected.
type quad struct {
	xs, ys [4]float64
	depth  float64
	fill   color.RGBA
	edge   color.RGBA
	order  int
}

// light is the one fixed light that shades the surface.
var light = func() [3]float64 {
	v := [3]float64{0.35, 0.45, 0.82}
	length := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	return [3]float64{v[0] / length, v[1] / length, v[2] / length}
}()

// surfaceRenderer draws Surfaces with one font face per pixel scale.
type surfaceRenderer struct {
	face func(size float64) font.Face
	math *surfaceMathCache
}

// render draws the Surface into a new image of the given pixel size, with
// the camera, text drawn at textScale.
func (r surfaceRenderer) render(s surfaceSpec, c camera, width, height int, textScale float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	top := 0.0
	titleFace := r.face(16 * textScale)
	if s.Title != "" {
		top = float64(titleFace.Metrics().Height.Ceil()) + 12*textScale
		r.drawText(img, titleFace, s.Title, float64(width)/2, 8*textScale+float64(titleFace.Metrics().Ascent.Ceil()), 0.5, color.RGBA{20, 20, 20, 255}, 16*textScale)
	}
	grid, lowZ, highZ := s.normalized()
	p := newProjection(c, float64(width), float64(height), top)
	line := math.Max(1, textScale)

	// The floor (z at its lowest) and its back walls frame the surface.
	corner := func(x, y, z float64) (float64, float64) {
		px, py, _ := p.at(point{x, y, z})
		return px, py
	}
	floor := []point{{-1, -1, -zHalf}, {1, -1, -zHalf}, {1, 1, -zHalf}, {-1, 1, -zHalf}}
	frame := color.RGBA{150, 150, 150, 255}
	drawFloor := func() {
		for index := range floor {
			a, b := floor[index], floor[(index+1)%len(floor)]
			x0, y0 := corner(a.x, a.y, a.z)
			x1, y1 := corner(b.x, b.y, b.z)
			strokeLine(img, x0, y0, x1, y1, line, frame)
		}
	}
	// The vertical z axis stands at the floor corner farthest to the left,
	// outside the surface, where it stays readable from every angle.
	side := floor[0]
	for _, candidate := range floor[1:] {
		cx, _ := corner(candidate.x, candidate.y, -zHalf)
		sx, _ := corner(side.x, side.y, -zHalf)
		if cx < sx || (cx == sx && dot(p.toward, candidate) < dot(p.toward, side)) {
			side = candidate
		}
	}
	if c.elevation >= 0 {
		drawFloor()
	}

	var quads []quad
	span := highZ - lowZ
	for j := 0; j+1 < len(grid); j++ {
		for i := 0; i+1 < len(grid[j]); i++ {
			corners := [4]point{grid[j][i], grid[j][i+1], grid[j+1][i+1], grid[j+1][i]}
			var q quad
			depth, height := 0.0, 0.0
			for index, value := range corners {
				q.xs[index], q.ys[index], _ = p.at(value)
				depth += dot(p.toward, value)
				height += (s.Z[j+index/2][i+((index+1)/2)%2] - lowZ)
			}
			q.depth = depth / 4
			t := 0.5
			if span > 0 {
				t = height / 4 / span
			}
			// Shade by the cell's normal, from its diagonals, toward the
			// light and whichever side faces the viewer.
			ax, ay, az := corners[2].x-corners[0].x, corners[2].y-corners[0].y, corners[2].z-corners[0].z
			bx, by, bz := corners[3].x-corners[1].x, corners[3].y-corners[1].y, corners[3].z-corners[1].z
			nx, ny, nz := ay*bz-az*by, az*bx-ax*bz, ax*by-ay*bx
			length := math.Sqrt(nx*nx + ny*ny + nz*nz)
			shade := 0.85
			if length > 0 {
				shade = 0.55 + 0.45*math.Abs((nx*light[0]+ny*light[1]+nz*light[2])/length)
			}
			base := heightColor(t)
			q.fill = toColor(base, shade)
			q.edge = toColor(base, shade*0.72)
			if s.Wireframe {
				q.edge = toColor(base, 0.8)
			}
			q.order = j*len(grid[j]) + i
			quads = append(quads, q)
		}
	}
	// Back to front; equal depths keep grid order, so drawing is
	// deterministic.
	sort.SliceStable(quads, func(a, b int) bool { return quads[a].depth < quads[b].depth })
	for _, q := range quads {
		if !s.Wireframe {
			fillTriangle(img, q.xs[0], q.ys[0], q.xs[1], q.ys[1], q.xs[2], q.ys[2], q.fill)
			fillTriangle(img, q.xs[0], q.ys[0], q.xs[2], q.ys[2], q.xs[3], q.ys[3], q.fill)
		}
		width := line * 0.6
		if s.Wireframe {
			width = line
		}
		for index := range 4 {
			next := (index + 1) % 4
			strokeLine(img, q.xs[index], q.ys[index], q.xs[next], q.ys[next], width, q.edge)
		}
	}
	if c.elevation < 0 {
		drawFloor()
	}
	axisBottomX, axisBottomY := corner(side.x, side.y, -zHalf)
	axisTopX, axisTopY := corner(side.x, side.y, zHalf)
	strokeLine(img, axisBottomX, axisBottomY, axisTopX, axisTopY, line, frame)

	// Axis names and ranges sit beside the floor's edges and the z axis.
	labelFace := r.face(12 * textScale)
	ink := color.RGBA{40, 40, 40, 255}
	label := func(text string, q point) {
		x, y, _ := p.at(q)
		r.drawText(img, labelFace, text, x, y, 0.5, ink, 12*textScale)
	}
	outside := 1.22
	// Named categories sit where a numeric axis showed only its two ends,
	// so the axis title steps further out to clear the row of names, and
	// the names themselves step back from the corner the two edges share.
	xTicks, yTicks := axisTicks(s.X, s.XCategories), axisTicks(s.Y, s.YCategories)
	xTitle, yTitle := outside, outside
	if len(xTicks) > 2 {
		xTitle = outside + 0.34
	}
	if len(yTicks) > 2 {
		yTitle = outside + 0.34
	}
	label(s.XLabel, point{0, -xTitle, -zHalf})
	for _, tick := range xTicks {
		label(tick.text, point{tick.at, -outside + 0.1, -zHalf})
	}
	label(s.YLabel, point{yTitle, 0, -zHalf})
	for _, tick := range yTicks {
		label(tick.text, point{outside - 0.1, tick.at, -zHalf})
	}
	gap := 8 * textScale
	ascent := float64(labelFace.Metrics().Ascent.Ceil())
	// The lowest value sits just above the floor, clear of the floor's own
	// range labels at the same corner.
	drawText(img, labelFace, formatNumber(lowZ), axisBottomX-gap, axisBottomY-ascent*0.6, 1, ink)
	drawText(img, labelFace, formatNumber(highZ), axisTopX-gap, axisTopY+ascent/2, 1, ink)
	r.drawText(img, labelFace, s.ZLabel, (axisBottomX+axisTopX)/2-gap, (axisBottomY+axisTopY)/2+ascent/2, 1, ink, 12*textScale)
	return img
}

// axisTick is one label beside a floor edge: its text, and where it sits on
// that edge in the drawing's -1..1 coordinates.
type axisTick struct {
	at   float64
	text string
}

// maxDrawnCategories is how many category labels fit along one floor edge
// before they would run into each other. Past it only the two ends are
// named, which is what a numeric axis has always shown.
const maxDrawnCategories = 8

// axisTicks chooses the labels for one axis. Without categories it is the
// first and last coordinate as numbers, exactly as before v2.2. With a
// short list of categories every one is drawn, in place, which is the whole
// point of naming them: a five-course axis reads as five courses.
func axisTicks(coordinates []float64, categories []string) []axisTick {
	if len(categories) != len(coordinates) || len(coordinates) == 0 {
		return []axisTick{
			{-1, formatNumber(coordinates[0])},
			{1, formatNumber(coordinates[len(coordinates)-1])},
		}
	}
	if len(categories) > maxDrawnCategories {
		return []axisTick{{-1, categories[0]}, {1, categories[len(categories)-1]}}
	}
	// The names are drawn across a slightly shorter span than the surface
	// itself, so the first and last of them step back from the corner the
	// two floor edges share instead of colliding there.
	const span = 0.88
	ticks := make([]axisTick, len(categories))
	low, high := coordinates[0], coordinates[len(coordinates)-1]
	for index, text := range categories {
		at := -span
		if high > low {
			at = (2*(coordinates[index]-low)/(high-low) - 1) * span
		}
		ticks[index] = axisTick{at: at, text: text}
	}
	return ticks
}

// formatNumber shows an axis bound briefly.
func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', 4, 64)
}

// drawText draws one line with its anchor at x (0 left, 0.5 center) and
// baseline at y.
func drawText(img *image.RGBA, face font.Face, text string, x, y, anchor float64, c color.Color) {
	if text == "" {
		return
	}
	width := font.MeasureString(face, text).Ceil()
	drawer := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face,
		Dot: fixed.P(int(math.Round(x-anchor*float64(width))), int(math.Round(y)))}
	drawer.DrawString(text)
}

func (r surfaceRenderer) drawText(img *image.RGBA, face font.Face, text string, x, y, anchor float64, c color.Color, size float64) {
	if r.math != nil {
		if rendered, ok := r.math.image(text, size, c); ok {
			width, height := rendered.Bounds().Dx(), rendered.Bounds().Dy()
			left := int(math.Round(x - anchor*float64(width)))
			top := int(math.Round(y - 0.75*float64(height)))
			draw.Draw(img, image.Rect(left, top, left+width, top+height), rendered, image.Point{}, draw.Over)
			return
		}
	}
	drawText(img, face, text, x, y, anchor, c)
}

// fillTriangle paints a triangle's pixels whose centers lie inside it.
func fillTriangle(img *image.RGBA, x0, y0, x1, y1, x2, y2 float64, c color.RGBA) {
	bounds := img.Bounds()
	minX := max(int(math.Floor(math.Min(x0, math.Min(x1, x2)))), bounds.Min.X)
	maxX := min(int(math.Ceil(math.Max(x0, math.Max(x1, x2)))), bounds.Max.X-1)
	minY := max(int(math.Floor(math.Min(y0, math.Min(y1, y2)))), bounds.Min.Y)
	maxY := min(int(math.Ceil(math.Max(y0, math.Max(y1, y2)))), bounds.Max.Y-1)
	area := (x1-x0)*(y2-y0) - (x2-x0)*(y1-y0)
	if area == 0 || minX > maxX || minY > maxY {
		return
	}
	for y := minY; y <= maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			w0 := (x1-px)*(y2-py) - (x2-px)*(y1-py)
			w1 := (x2-px)*(y0-py) - (x0-px)*(y2-py)
			w2 := (x0-px)*(y1-py) - (x1-px)*(y0-py)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// strokeLine paints a line segment of the given width by distance to the
// segment, with a soft edge.
func strokeLine(img *image.RGBA, x0, y0, x1, y1, width float64, c color.RGBA) {
	half := width / 2
	bounds := img.Bounds()
	minX := max(int(math.Floor(math.Min(x0, x1)-half-1)), bounds.Min.X)
	maxX := min(int(math.Ceil(math.Max(x0, x1)+half+1)), bounds.Max.X-1)
	minY := max(int(math.Floor(math.Min(y0, y1)-half-1)), bounds.Min.Y)
	maxY := min(int(math.Ceil(math.Max(y0, y1)+half+1)), bounds.Max.Y-1)
	dx, dy := x1-x0, y1-y0
	length := dx*dx + dy*dy
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			t := 0.0
			if length > 0 {
				t = math.Max(0, math.Min(1, ((px-x0)*dx+(py-y0)*dy)/length))
			}
			ex, ey := px-(x0+t*dx), py-(y0+t*dy)
			distance := math.Sqrt(ex*ex + ey*ey)
			coverage := math.Max(0, math.Min(1, half+0.5-distance))
			if coverage <= 0 {
				continue
			}
			blend(img, x, y, c, coverage)
		}
	}
}

func blend(img *image.RGBA, x, y int, c color.RGBA, coverage float64) {
	under := img.RGBAAt(x, y)
	mix := func(a, b uint8) uint8 {
		return uint8(math.Round(float64(a)*(1-coverage) + float64(b)*coverage))
	}
	img.SetRGBA(x, y, color.RGBA{mix(under.R, c.R), mix(under.G, c.G), mix(under.B, c.B), 255})
}

// renderCanonical is Surface.save's image: the initial camera, at the
// Surface's size in output pixels, drawn at twice the size and averaged
// down so edges are smooth.
func (r surfaceRenderer) renderCanonical(s surfaceSpec) *image.RGBA {
	width := int(math.Round(float64(s.Width) * pixelsPerUnit))
	height := int(math.Round(float64(s.Height) * pixelsPerUnit))
	large := r.render(s, initialCamera(), 2*width, 2*height, 2*pixelsPerUnit)
	return downsample(large, width, height)
}

// downsample averages each 2x2 block.
func downsample(source *image.RGBA, width, height int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			var r, g, b int
			for _, offset := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				c := source.RGBAAt(2*x+offset[0], 2*y+offset[1])
				r, g, b = r+int(c.R), g+int(c.G), b+int(c.B)
			}
			out.SetRGBA(x, y, color.RGBA{uint8((r + 2) / 4), uint8((g + 2) / 4), uint8((b + 2) / 4), 255})
		}
	}
	return out
}
