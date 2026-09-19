package main

import (
	"errors"
	"image"
	"io"
	"math"
	"path/filepath"
	"sync"
	"time"

	"ahdidentity"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Controls: the mouse wheel or a trackpad zooms around the pointer, dragging
// with the left button pans, Q and E turn the view a quarter turn counter-
// clockwise and clockwise, R resets, and Escape closes. The toolbar's
// buttons do the same, and Save saves the chart.
const (
	wheelStep = 1.15 // zoom factor per wheel notch
	hintTime  = 6 * time.Second
	hintText  = "Scroll: zoom · Drag: pan · Q/E: rotate · R: fit · Esc: close"
)

var backgroundColor = [4]float32{0.93, 0.93, 0.93, 1}

type viewer struct {
	chart    *ebiten.Image
	imageW   int
	imageH   int
	view     *view
	scale    float64 // device pixels per point
	started  time.Time
	dragging bool
	lastX    float64
	lastY    float64

	ready     func()
	readyOnce sync.Once

	hud      *hud
	hudLabel string
	hudImage *ebiten.Image
	hint     *ebiten.Image

	spec   viewSpec
	chrome *chrome
}

// perform runs a toolbar action.
func (v *viewer) perform(action string) {
	switch action {
	case actionSave:
		spec := v.spec
		v.chrome.startSave(func() (string, error) {
			path, err := chooseSavePath(spec.Dialog, "chart.png", []string{"png", "svg", "pdf"})
			if err != nil || path == "" {
				return "", err
			}
			return filepath.Base(path), saveChart(spec, path)
		})
	case actionZoomIn:
		v.view.zoomAt(buttonZoom, v.view.windowW/2, v.view.windowH/2)
	case actionZoomOut:
		v.view.zoomAt(1/buttonZoom, v.view.windowW/2, v.view.windowH/2)
	case actionRotateLeft:
		v.view.rotate(1)
	case actionRotateRight:
		v.view.rotate(-1)
	case actionFit:
		v.view.reset()
	}
}

func (v *viewer) Update() error {
	v.readyOnce.Do(func() {
		// The window is up: finish AhdCode's application identity and tell
		// the program that show() succeeded.
		ahdidentity.Apply()
		v.ready()
	})
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	x, y := v.cursor()
	action, overToolbar := v.chrome.clicked(x, y)
	v.perform(action)
	// The view lies below the toolbar.
	y -= toolbarHeight
	if _, dy := ebiten.Wheel(); dy != 0 && !math.IsNaN(dy) && !overToolbar {
		v.view.zoomAt(math.Pow(wheelStep, math.Max(-10, math.Min(10, dy))), x, y)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !overToolbar {
		v.dragging, v.lastX, v.lastY = true, x, y
	}
	if v.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			v.view.pan(x-v.lastX, y-v.lastY)
			v.lastX, v.lastY = x, y
		} else {
			v.dragging = false
		}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyQ):
		v.view.rotate(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyE):
		v.view.rotate(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyR):
		v.view.reset()
	}
	return nil
}

// cursor is the pointer position in points.
func (v *viewer) cursor() (float64, float64) {
	x, y := ebiten.CursorPosition()
	return float64(x) / v.scale, float64(y) / v.scale
}

func (v *viewer) Draw(screen *ebiten.Image) {
	screen.Fill(colorOf(backgroundColor))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(v.imageW)/2, -float64(v.imageH)/2)
	op.GeoM.Rotate(-float64(v.view.quarters) * math.Pi / 2)
	op.GeoM.Scale(v.view.zoom, v.view.zoom)
	op.GeoM.Translate(v.view.centerX, v.view.centerY+toolbarHeight)
	op.GeoM.Scale(v.scale, v.scale)
	if v.view.zoom*v.scale >= 2 {
		op.Filter = ebiten.FilterNearest
	} else {
		op.Filter = ebiten.FilterLinear
	}
	screen.DrawImage(v.chart, op)
	v.drawHUD(screen)
	v.chrome.draw(screen, v.scale)
}

// Layout works in device pixels, so the chart is drawn sharp on high-density
// displays; the view itself works in points.
func (v *viewer) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := ebiten.Monitor().DeviceScaleFactor()
	if scale <= 0 || math.IsNaN(scale) {
		scale = 1
	}
	v.scale = scale
	v.view.resize(outsideWidth, max(outsideHeight-toolbarHeight, 1))
	return int(math.Ceil(float64(outsideWidth) * scale)), int(math.Ceil(float64(outsideHeight) * scale))
}

// runWindow shows the chart until the user closes the viewer. It answers the
// handshake once the window's first frame runs, or with an error if no
// window could be opened.
func runWindow(img image.Image, spec viewSpec, out io.Writer) error {
	bounds := img.Bounds()
	boundW, boundH := 1400, 1000
	if monitorW, monitorH := ebiten.Monitor().Size(); monitorW > 0 && monitorH > 0 {
		boundW, boundH = monitorW*9/10, monitorH*9/10
	}
	windowW, windowH := initialWindow(bounds.Dx(), bounds.Dy(), boundW, boundH-toolbarHeight)
	answered := false
	v := &viewer{imageW: bounds.Dx(), imageH: bounds.Dy(), view: newView(bounds.Dx(), bounds.Dy(), windowW, windowH),
		scale: 1, started: time.Now(), hud: newHUD(), spec: spec}
	v.chrome = newChrome(modeChart, v.hud)
	title := spec.Title
	v.ready = func() {
		answered = true
		answer(out, reply{Ready: true})
	}
	ebiten.SetWindowTitle(windowTitle(title))
	ebiten.SetWindowSize(windowW, windowH+toolbarHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(320, 240, -1, -1)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetTPS(60)
	ahdidentity.Prepare()
	err := runGame(v, img)
	if !answered {
		if err == nil {
			err = errors.New("the viewer window closed before it was shown")
		}
		return errors.New("could not open the Plot viewer window: no usable display was found")
	}
	return nil
}

// runGame runs the window loop and turns a library panic into an error.
func runGame(v *viewer, img image.Image) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = errors.New("the window library stopped")
		}
	}()
	v.chart = ebiten.NewImageFromImage(img)
	return ebiten.RunGame(v)
}

// windowTitle names the viewer after the application: AhdCode's own, or a
// packaged application's.
func windowTitle(title string) string {
	name := ahdidentity.Name() + " Plot"
	if title != "" {
		name += " — " + title
	}
	return name
}
