package main

import (
	"math"
	"testing"
)

func approx(t *testing.T, what string, have, want float64) {
	t.Helper()
	if math.IsNaN(have) || math.IsInf(have, 0) || math.Abs(have-want) > 1e-9 {
		t.Fatalf("%s: have %v, want %v", what, have, want)
	}
}

func TestInitialFitKeepsTheAspectRatioAndCenters(t *testing.T) {
	v := newView(1600, 1200, 800, 800)
	approx(t, "fit zoom", v.zoom, 0.5)
	approx(t, "center x", v.centerX, 400)
	approx(t, "center y", v.centerY, 400)
	if v.rotationDegrees() != 0 || v.zoomPercent() != 50 {
		t.Fatal("initial HUD values")
	}
	// A small image is shown at its natural size, not enlarged.
	approx(t, "tiny image fit", newView(40, 30, 800, 600).zoom, 1)
	// A large Figure fits too.
	approx(t, "large figure fit", newView(1500, 1200, 750, 600).zoom, 0.5)
}

func TestZoomAtTheCenterAndAtThePointer(t *testing.T) {
	v := newView(800, 600, 800, 600)
	v.zoomAt(2, 400, 300)
	approx(t, "zoom", v.zoom, 2)
	approx(t, "center stays", v.centerX, 400)
	// Zooming at a point keeps the chart point under it in place.
	v = newView(800, 600, 800, 600)
	v.zoomAt(2, 600, 450)
	pointBefore := (600.0 - 400.0) / 1.0
	pointAfter := (600.0 - v.centerX) / v.zoom
	approx(t, "same chart point under the pointer", pointAfter, pointBefore)
	zoomed := v.zoom
	v.zoomAt(0.5, 600, 450)
	approx(t, "zoom out", v.zoom, zoomed*0.5)
}

func TestZoomBounds(t *testing.T) {
	v := newView(800, 600, 800, 600)
	for range 100 {
		v.zoomAt(1.5, 100, 100)
	}
	approx(t, "maximum", v.zoom, maxZoom)
	for range 100 {
		v.zoomAt(0.5, 100, 100)
	}
	approx(t, "minimum", v.zoom, v.fit()*minZoomOfFit)
	before := v.zoom
	for _, bad := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		v.zoomAt(bad, 10, 10)
	}
	approx(t, "bad factors ignored", v.zoom, before)
}

func TestPanStaysReachable(t *testing.T) {
	v := newView(800, 600, 800, 600)
	v.pan(100, 100)
	approx(t, "a fitting image stays centered x", v.centerX, 400)
	v.zoomAt(4, 400, 300)
	v.pan(50, -30)
	approx(t, "pan x", v.centerX, 450)
	approx(t, "pan y", v.centerY, 270)
	v.pan(1e9, 1e9)
	// The image (3200x2400 drawn) still covers the window.
	approx(t, "pan clamp x", v.centerX, 1600)
	approx(t, "pan clamp y", v.centerY, 1200)
	v.pan(-1e9, -1e9)
	approx(t, "pan clamp x low", v.centerX, 800-1600)
	v.pan(math.NaN(), math.Inf(1))
	approx(t, "bad deltas ignored", v.centerX, 800-1600)
}

func TestRotationQuarterTurns(t *testing.T) {
	v := newView(800, 400, 800, 800)
	approx(t, "landscape fit", v.zoom, 1)
	v.rotate(1)
	if v.rotationDegrees() != 90 {
		t.Fatal("Q turns 90")
	}
	w, h := v.turned()
	approx(t, "turned width", w, 400)
	approx(t, "turned height", h, 800)
	approx(t, "fit uses swapped bounds", v.zoom, 1)
	v.rotate(1)
	v.rotate(1)
	if v.rotationDegrees() != 270 {
		t.Fatal("three Q turns")
	}
	v.rotate(1)
	if v.rotationDegrees() != 0 {
		t.Fatal("four turns are the canonical 0")
	}
	v.rotate(-1)
	if v.rotationDegrees() != 270 {
		t.Fatal("E turns clockwise")
	}
	// Swapped bounds matter when they limit the fit.
	tall := newView(1200, 600, 600, 600)
	approx(t, "wide fit", tall.zoom, 0.5)
	tall.rotate(-1)
	approx(t, "turned fit", tall.zoom, 0.5)
	approx(t, "turned drawn height fits", 1200*tall.zoom, 600)
}

func TestZoomAndPanAfterRotation(t *testing.T) {
	v := newView(800, 400, 800, 800)
	v.rotate(1)
	v.zoomAt(4, 400, 400)
	v.pan(0, 1e9)
	// Turned, the image is 400 wide and 800 tall, drawn 4x.
	approx(t, "clamped along the turned height", v.centerY, 1600)
	approx(t, "clamped along the turned width", v.centerX, 400)
}

func TestResetAfterAnything(t *testing.T) {
	v := newView(800, 600, 800, 600)
	v.rotate(-1)
	v.zoomAt(3, 10, 10)
	v.pan(-200, 50)
	v.reset()
	if v.rotationDegrees() != 0 || v.touched {
		t.Fatal("reset rotation")
	}
	approx(t, "reset zoom", v.zoom, 1)
	approx(t, "reset center x", v.centerX, 400)
	approx(t, "reset center y", v.centerY, 300)
}

func TestResizeRefitsAnUntouchedView(t *testing.T) {
	v := newView(800, 600, 800, 600)
	v.resize(400, 300)
	approx(t, "refit", v.zoom, 0.5)
	v.zoomAt(2, 200, 150)
	v.resize(800, 600)
	approx(t, "a zoomed view keeps its zoom", v.zoom, 1)
}
