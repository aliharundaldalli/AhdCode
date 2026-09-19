package main

import (
	"bytes"
	"encoding/json"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func gridSurface(n int, f func(x, y float64) float64) surfaceSpec {
	s := surfaceSpec{Width: 300, Height: 225, XLabel: "x", YLabel: "y", ZLabel: "z"}
	for i := range n {
		s.X = append(s.X, -1+2*float64(i)/float64(n-1))
		s.Y = append(s.Y, -1+2*float64(i)/float64(n-1))
	}
	for _, y := range s.Y {
		row := make([]float64, n)
		for i, x := range s.X {
			row[i] = f(x, y)
		}
		s.Z = append(s.Z, row)
	}
	return s
}

func saddle(x, y float64) float64 { return x*x - y*y }
func plane(x, y float64) float64  { return 0.5*x + 0.25*y }

func TestSurfaceValidation(t *testing.T) {
	good := gridSurface(4, saddle)
	if err := good.validate(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*surfaceSpec){
		"one x":          func(s *surfaceSpec) { s.X = s.X[:1]; s.Z = [][]float64{{0}, {0}, {0}, {0}} },
		"rows":           func(s *surfaceSpec) { s.Z = s.Z[:3] },
		"columns":        func(s *surfaceSpec) { s.Z[0] = s.Z[0][:3] },
		"NaN":            func(s *surfaceSpec) { s.Z[1][1] = math.NaN() },
		"Inf":            func(s *surfaceSpec) { s.Z[1][1] = math.Inf(1) },
		"decreasing x":   func(s *surfaceSpec) { s.X[1], s.X[2] = s.X[2], s.X[1] },
		"too wide":       func(s *surfaceSpec) { s.Width = maxSurfaceSide + 1 },
		"too many x":     func(s *surfaceSpec) { s.X = make([]float64, maxGrid+1) },
		"invalid label":  func(s *surfaceSpec) { s.ZLabel = string([]byte{0xff}) },
		"repeated y":     func(s *surfaceSpec) { s.Y[3] = s.Y[2] },
		"infinite x":     func(s *surfaceSpec) { s.X[3] = math.Inf(1) },
		"zero height":    func(s *surfaceSpec) { s.Height = 0 },
		"nothing at all": func(s *surfaceSpec) { *s = surfaceSpec{} },
	} {
		bad := gridSurface(4, saddle)
		change(&bad)
		if bad.validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestCameraStaysBoundedAndResets(t *testing.T) {
	c := initialCamera()
	c.orbit(0, 500)
	if c.elevation != maxElevation {
		t.Fatalf("elevation passed the pole: %v", c.elevation)
	}
	c.orbit(0, -1000)
	if c.elevation != minElevation {
		t.Fatalf("elevation passed the lower pole: %v", c.elevation)
	}
	c.orbit(725, 0)
	if c.azimuth < -180 || c.azimuth >= 180 {
		t.Fatalf("azimuth did not wrap: %v", c.azimuth)
	}
	c.orbit(math.NaN(), math.Inf(1))
	c.zoomBy(1000)
	if c.zoom != maxSurfaceZoom {
		t.Fatalf("zoom %v", c.zoom)
	}
	c.zoomBy(1e-9)
	c.zoomBy(math.NaN())
	c.zoomBy(-2)
	if c.zoom != minSurfaceZoom {
		t.Fatalf("zoom %v", c.zoom)
	}
	c.panBy(1e9, -1e9, 800, 600)
	if c.panX != 800 || c.panY != -600 {
		t.Fatalf("pan %v %v", c.panX, c.panY)
	}
	for _, value := range []float64{c.azimuth, c.elevation, c.zoom, c.panX, c.panY} {
		if !finite(value) {
			t.Fatal("the camera reached a non-finite state")
		}
	}
	if initialCamera() != (camera{azimuth: initialAzimuth, elevation: initialElevation, zoom: 1}) {
		t.Fatal("reset camera")
	}
}

func TestSurfaceRenderingIsDeterministicAndShowsTheData(t *testing.T) {
	r := newSurfaceRenderer()
	encode := func(s surfaceSpec) []byte {
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, r.renderCanonical(s)); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}
	first, second := encode(gridSurface(20, saddle)), encode(gridSurface(20, saddle))
	if !bytes.Equal(first, second) {
		t.Fatal("the same Surface rendered two different images")
	}
	config, err := png.DecodeConfig(bytes.NewReader(first))
	if err != nil || config.Width != 400 || config.Height != 300 {
		t.Fatalf("300x225 units render at 4/3 pixels per unit: %v %v", config, err)
	}
	if bytes.Equal(first, encode(gridSurface(20, plane))) {
		t.Fatal("a saddle and a plane rendered the same image")
	}
	wire := gridSurface(20, saddle)
	wire.Wireframe = true
	if bytes.Equal(first, encode(wire)) {
		t.Fatal("wireframe changed nothing")
	}
	// A flat surface still draws: many pixels are not background.
	flat := r.renderCanonical(gridSurface(10, func(float64, float64) float64 { return 3 }))
	colored := 0
	for y := range flat.Bounds().Dy() {
		for x := range flat.Bounds().Dx() {
			if c := flat.RGBAAt(x, y); c.R != 255 || c.G != 255 || c.B != 255 {
				colored++
			}
		}
	}
	if colored < 5000 {
		t.Fatalf("a flat surface drew %d pixels", colored)
	}
	// The camera changes the view, never the canonical image.
	turned := r.render(gridSurface(20, saddle), camera{azimuth: 40, elevation: -30, zoom: 2}, 400, 300, 1)
	canonical := r.render(gridSurface(20, saddle), initialCamera(), 400, 300, 1)
	if bytes.Equal(turned.Pix, canonical.Pix) {
		t.Fatal("orbiting changed nothing")
	}
}

func TestSurfaceHeadlessCameraSaveAndRenderMode(t *testing.T) {
	surface := gridSurface(12, saddle)
	directory := t.TempDir()
	saved := filepath.Join(directory, "view.png")
	report := filepath.Join(directory, "report.json")
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS", "1")
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_REPORT", report)
	script, _ := json.Marshal([]action{{Action: "orbit", X: 30, Y: 200}, {Action: "zoom", Factor: 2}, {Action: "pan", X: 10, Y: -5},
		{Action: "save", Path: saved}, {Action: "reset"}, {Action: "save", Path: filepath.Join(directory, "view.svg")}, {Action: "close"}})
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_SCRIPT", string(script))
	var out bytes.Buffer
	if err := run([]string{writeSpec(t, viewSpec{Version: specVersion, Mode: modeSurface, Surface: &surface})}, &out); err != nil {
		t.Fatal(err)
	}
	var states []state
	data, _ := os.ReadFile(report)
	if json.Unmarshal(data, &states) != nil || len(states) != 8 {
		t.Fatalf("report %s", data)
	}
	if states[1].Elevation != maxElevation || states[2].Zoom != 2 || states[3].PanX != 10 {
		t.Fatalf("camera %+v", states[1:4])
	}
	if states[4].Saved != saved || states[5].Zoom != 1 || states[5].Elevation != initialElevation || states[6].Error == "" || !states[7].Closed {
		t.Fatalf("save and reset %+v", states[4:])
	}
	// Saving from a turned view writes exactly the canonical image, which is
	// what render mode (Surface.save) writes.
	rendered := filepath.Join(directory, "render.png")
	out.Reset()
	if err := run([]string{writeSpec(t, viewSpec{Version: specVersion, Mode: modeRender, Surface: &surface, Output: rendered})}, &out); err != nil {
		t.Fatal(err)
	}
	fromView, _ := os.ReadFile(saved)
	fromRender, _ := os.ReadFile(rendered)
	if len(fromView) == 0 || !bytes.Equal(fromView, fromRender) {
		t.Fatal("a save from the viewer differs from Surface.save")
	}
}

func TestToolbarButtons(t *testing.T) {
	chart, surface := toolbarFor(modeChart), toolbarFor(modeSurface)
	want := []string{actionSave, actionZoomOut, actionZoomIn, actionRotateLeft, actionRotateRight, actionFit}
	if len(chart) != len(want) || len(surface) != 4 {
		t.Fatalf("buttons %v %v", chart, surface)
	}
	for index, action := range want {
		if chart[index].action != action || chart[index].tip == "" {
			t.Fatalf("chart button %d = %+v", index, chart[index])
		}
	}
	for index := range chart {
		center := float64(buttonMargin+index*(buttonSize+buttonGap)) + buttonSize/2
		if buttonAt(chart, center, toolbarHeight/2) != index {
			t.Fatalf("button %d not hit", index)
		}
	}
	if buttonAt(chart, 2, toolbarHeight/2) != -1 || buttonAt(chart, 20, toolbarHeight+5) != -1 || buttonAt(surface, 200, 20) != -1 {
		t.Fatal("a miss hit a button")
	}
}
