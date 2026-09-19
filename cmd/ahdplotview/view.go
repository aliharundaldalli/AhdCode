package main

import "math"

// view is the viewer's whole transform: which part of the chart image the
// window shows, how large, and turned by how many quarter turns. It is pure
// arithmetic in window points (logical pixels), so every rule is tested
// without a display. Nothing here touches the chart or its data: rotating,
// zooming, and panning change only what the window shows.
type view struct {
	imageW, imageH   float64 // the rendered chart, in image pixels
	windowW, windowH float64 // the window, in points

	zoom             float64 // points per image pixel
	centerX, centerY float64 // where the image's center is drawn
	quarters         int     // counter-clockwise quarter turns, 0..3
	touched          bool    // the user zoomed or panned since the last fit
}

// Zoom bounds: a quarter of the fitted size, and 1600%.
const (
	minZoomOfFit = 0.25
	maxZoom      = 16.0
)

func newView(imageW, imageH, windowW, windowH int) *view {
	v := &view{imageW: float64(imageW), imageH: float64(imageH), windowW: float64(windowW), windowH: float64(windowH)}
	v.reset()
	return v
}

// turned is the image's size as drawn, with width and height swapped after
// a quarter turn.
func (v *view) turned() (float64, float64) {
	if v.quarters%2 == 1 {
		return v.imageH, v.imageW
	}
	return v.imageW, v.imageH
}

// fit is the zoom that shows the whole turned image, never enlarged past
// its natural size.
func (v *view) fit() float64 {
	w, h := v.turned()
	return math.Min(1, math.Min(v.windowW/w, v.windowH/h))
}

func (v *view) minZoom() float64 { return v.fit() * minZoomOfFit }

// reset returns to the canonical view: no turn, fitted, centered.
func (v *view) reset() {
	v.quarters = 0
	v.zoom = v.fit()
	v.centerX, v.centerY = v.windowW/2, v.windowH/2
	v.touched = false
}

// zoomAt scales the view by factor around the window point (x, y), which
// keeps showing the same part of the chart.
func (v *view) zoomAt(factor, x, y float64) {
	if !(factor > 0) || math.IsInf(factor, 0) {
		return
	}
	next := math.Min(maxZoom, math.Max(v.minZoom(), v.zoom*factor))
	if next == v.zoom {
		return
	}
	v.centerX = x - (x-v.centerX)*next/v.zoom
	v.centerY = y - (y-v.centerY)*next/v.zoom
	v.zoom = next
	v.touched = true
	v.clamp()
}

// pan moves the image by (dx, dy) points.
func (v *view) pan(dx, dy float64) {
	if math.IsNaN(dx) || math.IsNaN(dy) || math.IsInf(dx, 0) || math.IsInf(dy, 0) {
		return
	}
	v.centerX += dx
	v.centerY += dy
	v.touched = true
	v.clamp()
}

// rotate turns the view a quarter turn: +1 counter-clockwise (Q), -1
// clockwise (E). Four turns are the canonical 0.
func (v *view) rotate(direction int) {
	v.quarters = ((v.quarters+direction)%4 + 4) % 4
	if !v.touched {
		v.zoom = v.fit()
	}
	v.zoom = math.Min(maxZoom, math.Max(v.minZoom(), v.zoom))
	v.clamp()
}

// resize follows the window. A view the user has not zoomed or panned
// keeps fitting the new size.
func (v *view) resize(windowW, windowH int) {
	if windowW < 1 || windowH < 1 || (float64(windowW) == v.windowW && float64(windowH) == v.windowH) {
		return
	}
	v.windowW, v.windowH = float64(windowW), float64(windowH)
	if !v.touched {
		v.zoom = v.fit()
		v.centerX, v.centerY = v.windowW/2, v.windowH/2
		return
	}
	v.zoom = math.Min(maxZoom, math.Max(v.minZoom(), v.zoom))
	v.clamp()
}

// clamp keeps the image reachable. On an axis where it is smaller than the
// window it is centered; on an axis where it is larger, it always covers the
// window, so panning can never lose it.
func (v *view) clamp() {
	w, h := v.turned()
	v.centerX = clampAxis(v.centerX, w*v.zoom, v.windowW)
	v.centerY = clampAxis(v.centerY, h*v.zoom, v.windowH)
}

func clampAxis(center, drawn, window float64) float64 {
	if drawn <= window {
		return window / 2
	}
	return math.Min(drawn/2, math.Max(window-drawn/2, center))
}

// rotationDegrees is the turn shown in the HUD, counter-clockwise.
func (v *view) rotationDegrees() int { return v.quarters * 90 }

// zoomPercent is the zoom shown in the HUD.
func (v *view) zoomPercent() int { return int(math.Round(v.zoom * 100)) }
