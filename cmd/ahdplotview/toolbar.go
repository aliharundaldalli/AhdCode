package main

import (
	"image/color"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// The toolbar is the viewer's own, private chrome: a row of icon buttons
// along the top of the window, each with a tooltip. It is not a widget that
// AhdCode programs can use. Every button does exactly what a key or the
// mouse already does, except Save.

// Toolbar actions.
const (
	actionSave        = "save"
	actionZoomOut     = "zoomOut"
	actionZoomIn      = "zoomIn"
	actionRotateLeft  = "rotateLeft"
	actionRotateRight = "rotateRight"
	actionFit         = "reset"
)

// Toolbar metrics, in points.
const (
	toolbarHeight = 40
	buttonSize    = 30
	buttonGap     = 6
	buttonMargin  = 8
	buttonZoom    = 1.25 // one Zoom In or Zoom Out click
	statusTime    = 5 * time.Second
)

type toolButton struct {
	action string
	tip    string
}

// toolbarFor lists a mode's buttons, left to right.
func toolbarFor(mode string) []toolButton {
	if mode == modeSurface {
		return []toolButton{{actionSave, "Save (PNG)"}, {actionZoomOut, "Zoom Out"}, {actionZoomIn, "Zoom In"},
			{actionFit, "Reset View (R)"}}
	}
	return []toolButton{{actionSave, "Save (PNG, SVG, PDF)"}, {actionZoomOut, "Zoom Out"}, {actionZoomIn, "Zoom In"},
		{actionRotateLeft, "Rotate Left (Q)"}, {actionRotateRight, "Rotate Right (E)"}, {actionFit, "Fit (R)"}}
}

// buttonAt is the button under a point in points, or -1.
func buttonAt(buttons []toolButton, x, y float64) int {
	if y < (toolbarHeight-buttonSize)/2 || y >= (toolbarHeight+buttonSize)/2 {
		return -1
	}
	for index := range buttons {
		left := float64(buttonMargin + index*(buttonSize+buttonGap))
		if x >= left && x < left+buttonSize {
			return index
		}
	}
	return -1
}

// chrome is the toolbar's state: the button under the pointer, the save in
// progress, and the status line.
type chrome struct {
	buttons []toolButton
	hover   int
	saving  atomic.Bool

	mu        sync.Mutex
	status    string
	statusAt  time.Time
	hud       *hud
	tipImage  *ebiten.Image
	tipLabel  string
	statImage *ebiten.Image
	statLabel string
}

func newChrome(mode string, h *hud) *chrome {
	return &chrome{buttons: toolbarFor(mode), hover: -1, hud: h}
}

func (c *chrome) setStatus(text string) {
	c.mu.Lock()
	c.status, c.statusAt = text, time.Now()
	c.mu.Unlock()
}

// clicked reports the action of a toolbar button clicked this frame, and
// whether the pointer is over the toolbar, where no drag starts.
func (c *chrome) clicked(x, y float64) (string, bool) {
	c.hover = buttonAt(c.buttons, x, y)
	over := y < toolbarHeight
	if c.hover >= 0 && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return c.buttons[c.hover].action, over
	}
	return "", over
}

// startSave runs a save in the background: the dialog and the renderer are
// separate processes, and the window keeps drawing meanwhile. One save runs
// at a time.
func (c *chrome) startSave(save func() (string, error)) {
	if !c.saving.CompareAndSwap(false, true) {
		return
	}
	c.setStatus("Saving…")
	go func() {
		defer c.saving.Store(false)
		saved, err := save()
		switch {
		case err != nil:
			c.setStatus(err.Error())
		case saved == "":
			c.setStatus("")
		default:
			c.setStatus("Saved " + saved)
		}
	}()
}

var (
	toolbarColor = color.NRGBA{248, 248, 248, 255}
	toolbarLine  = color.NRGBA{205, 205, 205, 255}
	hoverColor   = color.NRGBA{222, 230, 242, 255}
	iconColor    = color.NRGBA{45, 45, 45, 255}
)

// draw paints the toolbar at the top of the screen, at scale device pixels
// per point.
func (c *chrome) draw(screen *ebiten.Image, scale float64) {
	width := float32(screen.Bounds().Dx())
	s := float32(scale)
	vector.FillRect(screen, 0, 0, width, toolbarHeight*s, toolbarColor, false)
	vector.FillRect(screen, 0, toolbarHeight*s-s, width, s, toolbarLine, false)
	for index, button := range c.buttons {
		left := float32(buttonMargin+index*(buttonSize+buttonGap)) * s
		top := float32(toolbarHeight-buttonSize) / 2 * s
		if index == c.hover {
			vector.FillRect(screen, left, top, buttonSize*s, buttonSize*s, hoverColor, true)
		}
		drawIcon(screen, button.action, left+buttonSize*s/2, top+buttonSize*s/2, s)
	}
	if c.hover >= 0 {
		tip := c.buttons[c.hover].tip
		if tip != c.tipLabel || c.tipImage == nil {
			if c.tipImage != nil {
				c.tipImage.Deallocate()
			}
			c.tipImage, c.tipLabel = c.hud.label(tip, scale), tip
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(buttonMargin+c.hover*(buttonSize+buttonGap))*scale, (toolbarHeight+4)*scale)
		screen.DrawImage(c.tipImage, op)
	}
	c.mu.Lock()
	status := c.status
	if time.Since(c.statusAt) > statusTime && !c.saving.Load() {
		status = ""
	}
	c.mu.Unlock()
	if status == "" {
		return
	}
	if status != c.statLabel || c.statImage == nil {
		if c.statImage != nil {
			c.statImage.Deallocate()
		}
		c.statImage, c.statLabel = c.hud.label(status, scale), status
	}
	left := float64(buttonMargin+len(c.buttons)*(buttonSize+buttonGap)+buttonGap) * scale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(left, (toolbarHeight*scale-float64(c.statImage.Bounds().Dy()))/2)
	screen.DrawImage(c.statImage, op)
}

// drawIcon draws one toolbar icon centered at (cx, cy), about 18 points
// across.
func drawIcon(screen *ebiten.Image, action string, cx, cy, s float32) {
	stroke := 1.8 * s
	line := func(x0, y0, x1, y1 float32) {
		vector.StrokeLine(screen, cx+x0*s, cy+y0*s, cx+x1*s, cy+y1*s, stroke, iconColor, true)
	}
	arc := func(clockwise bool) {
		var path vector.Path
		start, end := float32(math.Pi*0.15), float32(math.Pi*1.75)
		direction := vector.CounterClockwise
		if clockwise {
			start, end = float32(math.Pi*0.85), float32(-math.Pi*0.75)
			direction = vector.Clockwise
		}
		path.Arc(cx, cy, 7*s, start, end, direction)
		vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: stroke}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(iconColor)})
		// The arrow head at the arc's end.
		ex, ey := cx+7*s*float32(math.Cos(float64(end))), cy+7*s*float32(math.Sin(float64(end)))
		size := 3.5 * s
		if clockwise {
			vector.StrokeLine(screen, ex, ey, ex-size, ey-size*0.2, stroke, iconColor, true)
			vector.StrokeLine(screen, ex, ey, ex+size*0.2, ey-size, stroke, iconColor, true)
		} else {
			vector.StrokeLine(screen, ex, ey, ex+size, ey-size*0.2, stroke, iconColor, true)
			vector.StrokeLine(screen, ex, ey, ex-size*0.2, ey-size, stroke, iconColor, true)
		}
	}
	switch action {
	case actionSave:
		line(0, -8, 0, 3)
		line(-4, -1, 0, 3)
		line(4, -1, 0, 3)
		line(-8, 3, -8, 8)
		line(-8, 8, 8, 8)
		line(8, 8, 8, 3)
	case actionZoomOut, actionZoomIn:
		vector.StrokeCircle(screen, cx-2*s, cy-2*s, 6*s, stroke, iconColor, true)
		line(2.5, 2.5, 8, 8)
		line(-5, -2, 1, -2)
		if action == actionZoomIn {
			line(-2, -5, -2, 1)
		}
	case actionRotateLeft:
		arc(false)
	case actionRotateRight:
		arc(true)
	case actionFit:
		for _, corner := range [][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			x, y := 8*corner[0], 8*corner[1]
			line(x, y, x-4*corner[0], y)
			line(x, y, x, y-4*corner[1])
		}
	}
}

func colorScale(c color.NRGBA) ebiten.ColorScale {
	var scale ebiten.ColorScale
	scale.ScaleWithColor(c)
	return scale
}

// faceCache holds Go Regular faces by pixel size.
type faceCache struct {
	mu    sync.Mutex
	font  *opentype.Font
	faces map[float64]font.Face
}

var sharedFaces = &faceCache{faces: map[float64]font.Face{}}

func (cache *faceCache) face(size float64) font.Face {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.font == nil {
		parsed, err := opentype.Parse(goregular.TTF)
		if err != nil {
			panic("the embedded Go Regular font does not parse")
		}
		cache.font = parsed
	}
	size = math.Round(size*4) / 4
	if face, ok := cache.faces[size]; ok {
		return face
	}
	face, err := opentype.NewFace(cache.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic("the embedded Go Regular font has no face")
	}
	cache.faces[size] = face
	return face
}

// newSurfaceRenderer draws with the shared faces.
func newSurfaceRenderer() surfaceRenderer {
	return surfaceRenderer{face: sharedFaces.face}
}
