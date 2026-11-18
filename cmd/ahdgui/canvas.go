package main

import (
	"errors"
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// An embedded Canvas uses the same coordinates as a Graphics Canvas: the
// origin is the center of the widget and +y points up. Pointer positions
// are converted from window pixels, after layout has placed the widget, so
// a scroll offset is already part of the widget's position.

const maxCanvasCommands = 4096

type canvasCommand struct {
	kind                 string
	x, y, x2, y2, radius float64
	strokeWidth          float64
	stroke, fill         color.NRGBA
	hasFill              bool
	text                 string
}

func (m *model) canvasPoint(target *widget, x, y int, outside bool) (float64, float64, bool) {
	if target == nil || target.kind != kindCanvas || target.w <= 0 || target.h <= 0 {
		return 0, 0, false
	}
	if !outside {
		if !m.shown(target.id) || !m.tabShown(target.id) || !m.insideScrolls(target.id, x, y) {
			return 0, 0, false
		}
		if x < target.x || y < target.y || x >= target.x+target.w || y >= target.y+target.h {
			return 0, 0, false
		}
	}
	gx := float64(x-target.x) - float64(target.w)/2
	gy := float64(target.h)/2 - float64(y-target.y)
	return gx, gy, true
}

// canvasCoords lays the window out and converts a window point. The point
// may lie outside the Canvas when outside is set, which a drag needs.
func (m *model) canvasCoords(id int64, x, y int, outside bool) (float64, float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.layout()
	target := m.widgets[id]
	if target == nil || target.disabled {
		return 0, 0, false
	}
	return m.canvasPoint(target, x, y, outside)
}

func (m *model) queuePointer(kind string, id int64, x, y float64) {
	if m.pointerReady && m.pointerKind == "move" && kind == "move" {
		m.pointerX, m.pointerY = x, y
		return
	}
	m.pointerKind, m.pointerID, m.pointerX, m.pointerY = kind, id, x, y
	m.pointerReady = true
}

func (m *model) takePointer() (string, int64, float64, float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.pointerReady {
		return "", 0, 0, 0, false
	}
	m.pointerReady = false
	return m.pointerKind, m.pointerID, m.pointerX, m.pointerY, true
}

func (m *model) hover(x, y int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.canvasPress != 0 {
		return
	}
	if m.hoverReady && x == m.hoverX && y == m.hoverY {
		return
	}
	m.hoverReady = true
	m.hoverX, m.hoverY = x, y
	m.layout()
	id := m.hit(x, y)
	if id == 0 || m.widgets[id].kind != kindCanvas || m.widgets[id].disabled {
		return
	}
	gx, gy, ok := m.canvasPoint(m.widgets[id], x, y, false)
	if ok {
		m.queuePointer("move", id, gx, gy)
	}
}

func (m *model) drawCanvas(r request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[r.Widget]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindCanvas {
		return errWrongKind
	}
	if r.Kind == "clear" {
		parsed, ok := parseColor(r.Color)
		if !ok {
			return errCanvasColor
		}
		target.background = &parsed
		target.commands = nil
		m.dirty = true
		return nil
	}
	if len(target.commands) >= maxCanvasCommands {
		return errCanvasFull
	}
	command := canvasCommand{kind: r.Kind, x: r.X, y: r.Y, x2: r.X2, y2: r.Y2, radius: r.Radius, strokeWidth: r.StrokeW, text: r.Text}
	parsed, ok := parseColor(r.Color)
	if !ok {
		return errCanvasColor
	}
	command.stroke = parsed
	if r.Fill != "" {
		filled, ok := parseColor(r.Fill)
		if !ok {
			return errCanvasColor
		}
		command.fill = filled
		command.hasFill = true
	}
	target.commands = append(target.commands, command)
	m.dirty = true
	return nil
}

var (
	errCanvasFull  = errors.New("a Canvas holds at most 4096 drawing commands; clear it first")
	errCanvasColor = errors.New("invalid color")
)

func drawCanvas(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle) {
	if clipped, ok := img.SubImage(box.Intersect(img.Bounds())).(*image.RGBA); ok {
		img = clipped
	}
	background := color.NRGBA{255, 255, 255, 255}
	if w.background != nil {
		background = *w.background
	}
	fill(img, box, background)
	for _, command := range w.commands {
		paintCanvas(img, face, f, w, box, command)
	}
}

func canvasPixel(w *widget, box image.Rectangle, gx, gy float64) (int, int) {
	sx := float64(box.Dx()) / float64(max(w.w, 1))
	sy := float64(box.Dy()) / float64(max(w.h, 1))
	px := box.Min.X + int(math.Round((gx+float64(w.w)/2)*sx))
	py := box.Min.Y + int(math.Round((float64(w.h)/2-gy)*sy))
	return px, py
}

func paintCanvas(img *image.RGBA, face font.Face, f frame, w *widget, box image.Rectangle, command canvasCommand) {
	switch command.kind {
	case "line":
		x0, y0 := canvasPixel(w, box, command.x, command.y)
		x1, y1 := canvasPixel(w, box, command.x2, command.y2)
		strokeLine(img, box, [2]int{x0, y0}, [2]int{x1, y1}, command.strokeWidth*f.scale, command.stroke)
	case "rectangle":
		x0, y0 := canvasPixel(w, box, command.x, command.y+command.y2)
		x1, y1 := canvasPixel(w, box, command.x+command.x2, command.y)
		rect := image.Rect(min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1)).Intersect(box)
		if command.hasFill {
			fill(img, rect, command.fill)
		}
		outline(img, rect, command.stroke, max(1, int(math.Round(command.strokeWidth*f.scale))))
	case "circle":
		cx, cy := canvasPixel(w, box, command.x, command.y)
		rx := command.radius * float64(box.Dx()) / float64(max(w.w, 1))
		ry := command.radius * float64(box.Dy()) / float64(max(w.h, 1))
		paintCircle(img, box, cx, cy, rx, ry, command.strokeWidth*f.scale, command.stroke, command.fill, command.hasFill)
	case "text":
		px, py := canvasPixel(w, box, command.x, command.y)
		sized := faceAt(f.scale * command.strokeWidth / fontSize)
		drawer := font.Drawer{Dst: img, Src: image.NewUniform(command.stroke), Face: sized, Dot: fixed.P(px, py)}
		drawer.DrawString(command.text)
		_ = face
	}
}

func strokeLine(img *image.RGBA, box image.Rectangle, from, to [2]int, width float64, color color.NRGBA) {
	if width < 1 {
		width = 1
	}
	radius := width / 2
	minX := min(from[0], to[0]) - int(math.Ceil(radius))
	maxX := max(from[0], to[0]) + int(math.Ceil(radius))
	minY := min(from[1], to[1]) - int(math.Ceil(radius))
	maxY := max(from[1], to[1]) + int(math.Ceil(radius))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !image.Pt(x, y).In(box) {
				continue
			}
			if distanceToSegment(float64(x), float64(y), float64(from[0]), float64(from[1]), float64(to[0]), float64(to[1])) <= radius {
				img.Set(x, y, color)
			}
		}
	}
}

func distanceToSegment(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	length := dx*dx + dy*dy
	if length == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / length
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

func paintCircle(img *image.RGBA, box image.Rectangle, cx, cy int, rx, ry, stroke float64, strokeColor, fillColor color.NRGBA, hasFill bool) {
	if rx < 0.5 || ry < 0.5 {
		return
	}
	reach := math.Max(rx, ry) + stroke
	for y := int(math.Floor(float64(cy) - reach)); y <= int(math.Ceil(float64(cy)+reach)); y++ {
		for x := int(math.Floor(float64(cx) - reach)); x <= int(math.Ceil(float64(cx)+reach)); x++ {
			if !image.Pt(x, y).In(box) {
				continue
			}
			nx := (float64(x) - float64(cx)) / rx
			ny := (float64(y) - float64(cy)) / ry
			distance := math.Hypot(nx, ny)
			if hasFill && distance <= 1 {
				img.Set(x, y, fillColor)
			}
			if math.Abs(distance-1)*math.Min(rx, ry) <= math.Max(stroke, 1)/2 {
				img.Set(x, y, strokeColor)
			}
		}
	}
}
