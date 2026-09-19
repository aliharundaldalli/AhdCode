package main

import (
	"errors"
	"io"
	"math"
	"path/filepath"
	"sync"
	"time"

	"ahdidentity"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The Surface viewer: dragging with the left button orbits, Shift+drag or
// dragging with the right button pans, the wheel zooms, R resets the camera,
// and Escape closes. The picture is drawn by the same software renderer as
// Surface.save, again whenever the camera or the window changes.

const (
	orbitPerPoint  = 0.4 // degrees of orbit per point dragged
	surfaceHint    = "Drag: orbit · Shift+drag: pan · Scroll: zoom · R: reset · Esc: close"
	surfaceMinSide = 320
)

type surfaceViewer struct {
	spec     viewSpec
	camera   camera
	renderer surfaceRenderer
	scale    float64
	width    int // points
	height   int // points, below the toolbar
	started  time.Time

	picture   *ebiten.Image
	dirty     bool
	dragging  bool
	panning   bool
	lastX     float64
	lastY     float64
	ready     func()
	readyOnce sync.Once

	hud    *hud
	hint   *ebiten.Image
	chrome *chrome
}

func (v *surfaceViewer) perform(action string) {
	switch action {
	case actionSave:
		surface := *v.spec.Surface
		dialog := v.spec.Dialog
		v.chrome.startSave(func() (string, error) {
			path, err := chooseSavePath(dialog, "surface.png", []string{"png"})
			if err != nil || path == "" {
				return "", err
			}
			return filepath.Base(path), saveSurface(surface, path)
		})
	case actionZoomIn:
		v.camera.zoomBy(buttonZoom)
	case actionZoomOut:
		v.camera.zoomBy(1 / buttonZoom)
	case actionFit:
		v.camera = initialCamera()
	default:
		return
	}
	v.dirty = true
}

func (v *surfaceViewer) Update() error {
	v.readyOnce.Do(func() {
		ahdidentity.Apply()
		v.ready()
	})
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	px, py := ebiten.CursorPosition()
	x, y := float64(px)/v.scale, float64(py)/v.scale
	action, overToolbar := v.chrome.clicked(x, y)
	v.perform(action)
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		v.perform(actionFit)
	}
	if _, dy := ebiten.Wheel(); dy != 0 && !math.IsNaN(dy) && !overToolbar {
		v.camera.zoomBy(math.Pow(wheelStep, math.Max(-10, math.Min(10, dy))))
		v.dirty = true
	}
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	switch {
	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !overToolbar:
		v.dragging, v.panning, v.lastX, v.lastY = true, shift, x, y
	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && !overToolbar:
		v.dragging, v.panning, v.lastX, v.lastY = true, true, x, y
	}
	if v.dragging {
		held := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
		if !held {
			v.dragging = false
		} else if dx, dy := x-v.lastX, y-v.lastY; dx != 0 || dy != 0 {
			if v.panning {
				v.camera.panBy(dx*v.scale, dy*v.scale, float64(v.width)*v.scale, float64(v.height)*v.scale)
			} else {
				v.camera.orbit(-dx*orbitPerPoint, dy*orbitPerPoint)
			}
			v.lastX, v.lastY = x, y
			v.dirty = true
		}
	}
	return nil
}

func (v *surfaceViewer) Draw(screen *ebiten.Image) {
	screen.Fill(colorOf([4]float32{1, 1, 1, 1}))
	width, height := int(math.Ceil(float64(v.width)*v.scale)), int(math.Ceil(float64(v.height)*v.scale))
	if v.dirty || v.picture == nil || v.picture.Bounds().Dx() != width || v.picture.Bounds().Dy() != height {
		img := v.renderer.render(*v.spec.Surface, v.camera, width, height, v.scale)
		if v.picture != nil {
			v.picture.Deallocate()
		}
		v.picture = ebiten.NewImageFromImage(img)
		v.dirty = false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, toolbarHeight*v.scale)
	screen.DrawImage(v.picture, op)
	if time.Since(v.started) < hintTime {
		if v.hint == nil {
			v.hint = v.hud.label(surfaceHint, v.scale)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate((float64(screen.Bounds().Dx())-float64(v.hint.Bounds().Dx()))/2, (toolbarHeight+10)*v.scale)
		screen.DrawImage(v.hint, op)
	} else if v.hint != nil {
		v.hint.Deallocate()
		v.hint = nil
	}
	v.chrome.draw(screen, v.scale)
}

func (v *surfaceViewer) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := ebiten.Monitor().DeviceScaleFactor()
	if scale <= 0 || math.IsNaN(scale) {
		scale = 1
	}
	if scale != v.scale {
		v.scale, v.dirty = scale, true
	}
	v.width, v.height = max(outsideWidth, 1), max(outsideHeight-toolbarHeight, 1)
	return int(math.Ceil(float64(outsideWidth) * scale)), int(math.Ceil(float64(outsideHeight) * scale))
}

// runSurfaceWindow shows the Surface until the user closes the viewer.
func runSurfaceWindow(spec viewSpec, out io.Writer) error {
	width := int(math.Round(float64(spec.Surface.Width) * pixelsPerUnit))
	height := int(math.Round(float64(spec.Surface.Height) * pixelsPerUnit))
	boundW, boundH := 1400, 1000
	if monitorW, monitorH := ebiten.Monitor().Size(); monitorW > 0 && monitorH > 0 {
		boundW, boundH = monitorW*9/10, monitorH*9/10
	}
	windowW, windowH := initialWindow(width, height, boundW, boundH-toolbarHeight)
	h := newHUD()
	answered := false
	v := &surfaceViewer{spec: spec, camera: initialCamera(), renderer: newSurfaceRenderer(), scale: 1,
		width: windowW, height: windowH, started: time.Now(), dirty: true, hud: h, chrome: newChrome(modeSurface, h)}
	v.ready = func() {
		answered = true
		answer(out, reply{Ready: true})
	}
	ebiten.SetWindowTitle(windowTitle(spec.Title))
	ebiten.SetWindowSize(windowW, windowH+toolbarHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(surfaceMinSide, surfaceMinSide*3/4, -1, -1)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetTPS(60)
	ahdidentity.Prepare()
	err := runSurfaceGame(v)
	if !answered {
		if err == nil {
			err = errors.New("the viewer window closed before it was shown")
		}
		return errors.New("could not open the Plot viewer window: no usable display was found")
	}
	return nil
}

func runSurfaceGame(v *surfaceViewer) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = errors.New("the window library stopped")
		}
	}()
	return ebiten.RunGame(v)
}
