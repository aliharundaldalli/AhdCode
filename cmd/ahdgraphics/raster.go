package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"golang.org/x/image/vector"
)

// The rasterizer is deterministic and needs no window or GPU: the window
// shows its image, and PNG export encodes the same rendering at scale 1.
// Canvas coordinates are Cartesian with the origin at the center and +y up;
// pixel coordinates have the origin at the top-left and +y down.

// bezierCircle is the control-point distance, relative to the radius, of the
// four-cubic circle approximation.
const bezierCircle = 0.5522847498307936

type point struct{ x, y float64 }

type pathOp struct {
	kind uint8 // 0 move, 1 line, 2 cube, 3 close
	p    [3]point
}

// shapePath collects the closed sub-paths of one primitive and their bounds.
// All sub-paths that add coverage share one orientation; holes use the
// opposite one, so overlapping parts of a stroke union instead of cancelling.
type shapePath struct {
	ops                    []pathOp
	minX, minY, maxX, maxY float64
}

func newShapePath() *shapePath {
	return &shapePath{minX: math.Inf(1), minY: math.Inf(1), maxX: math.Inf(-1), maxY: math.Inf(-1)}
}

func (s *shapePath) include(p point) {
	s.minX, s.maxX = math.Min(s.minX, p.x), math.Max(s.maxX, p.x)
	s.minY, s.maxY = math.Min(s.minY, p.y), math.Max(s.maxY, p.y)
}

func signedArea(points []point) float64 {
	area := 0.0
	for index := range points {
		next := points[(index+1)%len(points)]
		area += points[index].x*next.y - next.x*points[index].y
	}
	return area / 2
}

// polygon adds a closed polygon with positive (hole == false) or negative
// orientation.
func (s *shapePath) polygon(points []point, hole bool) {
	if len(points) < 3 {
		return
	}
	area := signedArea(points)
	if area == 0 {
		return
	}
	if (area < 0) != hole {
		reversed := make([]point, len(points))
		for index := range points {
			reversed[index] = points[len(points)-1-index]
		}
		points = reversed
	}
	s.ops = append(s.ops, pathOp{kind: 0, p: [3]point{points[0]}})
	s.include(points[0])
	for _, p := range points[1:] {
		s.ops = append(s.ops, pathOp{kind: 1, p: [3]point{p}})
		s.include(p)
	}
	s.ops = append(s.ops, pathOp{kind: 3})
}

// circle adds a closed circle; hole reverses its orientation.
func (s *shapePath) circle(center point, radius float64, hole bool) {
	if radius <= 0 {
		return
	}
	direction := 1.0
	if hole {
		direction = -1
	}
	at := func(angle float64) point {
		return point{center.x + radius*math.Cos(angle), center.y + radius*math.Sin(angle)}
	}
	tangent := func(angle float64) point {
		return point{-math.Sin(angle) * direction, math.Cos(angle) * direction}
	}
	start := at(0)
	s.ops = append(s.ops, pathOp{kind: 0, p: [3]point{start}})
	for quarter := 0; quarter < 4; quarter++ {
		from := direction * float64(quarter) * math.Pi / 2
		to := direction * float64(quarter+1) * math.Pi / 2
		p0, p3 := at(from), at(to)
		t0, t3 := tangent(from), tangent(to)
		p1 := point{p0.x + bezierCircle*radius*t0.x, p0.y + bezierCircle*radius*t0.y}
		p2 := point{p3.x - bezierCircle*radius*t3.x, p3.y - bezierCircle*radius*t3.y}
		s.ops = append(s.ops, pathOp{kind: 2, p: [3]point{p1, p2, p3}})
	}
	s.ops = append(s.ops, pathOp{kind: 3})
	s.include(point{center.x - radius, center.y - radius})
	s.include(point{center.x + radius, center.y + radius})
}

// fill rasterizes the path onto img inside the path's bounds only, so the
// cost of a primitive follows its own size rather than the Canvas size.
func (s *shapePath) fill(img *image.RGBA, value rgba) {
	if len(s.ops) == 0 {
		return
	}
	bounds := img.Bounds()
	x0 := max(int(math.Floor(s.minX)), bounds.Min.X)
	y0 := max(int(math.Floor(s.minY)), bounds.Min.Y)
	x1 := min(int(math.Ceil(s.maxX))+1, bounds.Max.X)
	y1 := min(int(math.Ceil(s.maxY))+1, bounds.Max.Y)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	rasterizer := vector.NewRasterizer(x1-x0, y1-y0)
	rasterizer.DrawOp = draw.Over
	local := func(p point) (float32, float32) { return float32(p.x - float64(x0)), float32(p.y - float64(y0)) }
	for _, op := range s.ops {
		switch op.kind {
		case 0:
			rasterizer.MoveTo(local(op.p[0]))
		case 1:
			rasterizer.LineTo(local(op.p[0]))
		case 2:
			ax, ay := local(op.p[0])
			bx, by := local(op.p[1])
			cx, cy := local(op.p[2])
			rasterizer.CubeTo(ax, ay, bx, by, cx, cy)
		case 3:
			rasterizer.ClosePath()
		}
	}
	source := image.NewUniform(color.NRGBA{R: value[0], G: value[1], B: value[2], A: value[3]})
	rasterizer.Draw(img, image.Rect(x0, y0, x1, y1), source, image.Point{})
}

// transform maps Canvas coordinates to pixel coordinates at one scale.
type transform struct {
	width, height float64
	scale         float64
}

func (t transform) pixel(x, y float64) point {
	return point{(x + t.width/2) * t.scale, (t.height/2 - y) * t.scale}
}

// clipSegment clips a pixel-space segment to a box with Liang-Barsky, so a
// line that runs far outside the Canvas never reaches the rasterizer with
// huge coordinates. It reports false when nothing of the segment is inside.
func clipSegment(a, b point, minX, minY, maxX, maxY float64) (point, point, bool) {
	t0, t1 := 0.0, 1.0
	dx, dy := b.x-a.x, b.y-a.y
	checks := [4][2]float64{{-dx, a.x - minX}, {dx, maxX - a.x}, {-dy, a.y - minY}, {dy, maxY - a.y}}
	for _, check := range checks {
		p, q := check[0], check[1]
		if p == 0 {
			if q < 0 {
				return a, b, false
			}
			continue
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return a, b, false
			}
			t0 = math.Max(t0, r)
		} else {
			if r < t0 {
				return a, b, false
			}
			t1 = math.Min(t1, r)
		}
	}
	return point{a.x + t0*dx, a.y + t0*dy}, point{a.x + t1*dx, a.y + t1*dy}, true
}

// drawCommand renders one primitive. A circle of radius 0 and a rectangle
// with a zero side draw nothing, which is also what SVG renders for them.
func drawCommand(img *image.RGBA, t transform, value command) {
	bounds := img.Bounds()
	halfWidth := value.width * t.scale / 2
	margin := halfWidth + 2
	minX, minY := float64(bounds.Min.X)-margin, float64(bounds.Min.Y)-margin
	maxX, maxY := float64(bounds.Max.X)+margin, float64(bounds.Max.Y)+margin

	switch value.kind {
	case lineCommand:
		a, b, visible := clipSegment(t.pixel(value.a, value.b), t.pixel(value.c, value.d), minX, minY, maxX, maxY)
		if !visible {
			return
		}
		shape := newShapePath()
		length := math.Hypot(b.x-a.x, b.y-a.y)
		if length > 1e-9 {
			nx, ny := -(b.y-a.y)/length*halfWidth, (b.x-a.x)/length*halfWidth
			shape.polygon([]point{{a.x + nx, a.y + ny}, {b.x + nx, b.y + ny}, {b.x - nx, b.y - ny}, {a.x - nx, a.y - ny}}, false)
		}
		// Round caps: each end is a disc of the stroke's width.
		shape.circle(a, halfWidth, false)
		shape.circle(b, halfWidth, false)
		shape.fill(img, value.stroke)

	case circleCommand:
		if value.c <= 0 {
			return
		}
		center := t.pixel(value.a, value.b)
		radius := value.c * t.scale
		if center.x+radius+halfWidth < minX || center.x-radius-halfWidth > maxX ||
			center.y+radius+halfWidth < minY || center.y-radius-halfWidth > maxY {
			return
		}
		if value.hasFill {
			body := newShapePath()
			body.circle(center, radius, false)
			body.fill(img, value.fill)
		}
		ring := newShapePath()
		ring.circle(center, radius+halfWidth, false)
		ring.circle(center, radius-halfWidth, true)
		ring.fill(img, value.stroke)

	case rectangleCommand:
		if value.c <= 0 || value.d <= 0 {
			return
		}
		topLeft := t.pixel(value.a, value.b+value.d)
		bottomRight := t.pixel(value.a+value.c, value.b)
		left, top := math.Max(topLeft.x, minX), math.Max(topLeft.y, minY)
		right, bottom := math.Min(bottomRight.x, maxX), math.Min(bottomRight.y, maxY)
		if left >= right || top >= bottom {
			return
		}
		rectangle := func(l, t, r, b float64) []point { return []point{{l, t}, {r, t}, {r, b}, {l, b}} }
		if value.hasFill {
			body := newShapePath()
			body.polygon(rectangle(left, top, right, bottom), false)
			body.fill(img, value.fill)
		}
		frame := newShapePath()
		frame.polygon(rectangle(left-halfWidth, top-halfWidth, right+halfWidth, bottom+halfWidth), false)
		if right-left > 2*halfWidth && bottom-top > 2*halfWidth {
			frame.polygon(rectangle(left+halfWidth, top+halfWidth, right-halfWidth, bottom-halfWidth), true)
		}
		frame.fill(img, value.stroke)
	}
}

// newImage allocates a Canvas image at a scale and paints its background.
func newImage(width, height int, scale float64, background rgba) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(width)*scale)), int(math.Ceil(float64(height)*scale))))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{R: background[0], G: background[1], B: background[2], A: background[3]}), image.Point{}, draw.Src)
	return img
}

// render draws a snapshot from scratch.
func render(value snapshot, scale float64) *image.RGBA {
	img := newImage(value.width, value.height, scale, value.background)
	t := transform{width: float64(value.width), height: float64(value.height), scale: scale}
	for _, drawn := range value.commands {
		drawCommand(img, t, drawn)
	}
	return img
}

// encodePNG renders a snapshot at one pixel per Canvas unit.
func encodePNG(value snapshot) ([]byte, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, render(value, 1)); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
