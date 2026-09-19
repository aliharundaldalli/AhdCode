package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The GUI draws itself: every widget is rendered into one RGBA image with the
// Go Regular font, which is embedded in this helper (see golang.org/x/image/
// font/gofont), so text looks the same on every computer and no system font
// is needed. The window only shows the image.

var (
	colorBackground  = color.RGBA{242, 242, 242, 255}
	colorText        = color.RGBA{28, 28, 28, 255}
	colorMuted       = color.RGBA{140, 140, 140, 255}
	colorBorder      = color.RGBA{150, 150, 150, 255}
	colorButton      = color.RGBA{226, 226, 226, 255}
	colorButtonPress = color.RGBA{198, 198, 198, 255}
	colorField       = color.RGBA{255, 255, 255, 255}
	colorFocus       = color.RGBA{0, 110, 220, 255}
	// colorDisabled fades a disabled widget: the default window color at 60%
	// opacity, painted over the finished widget.
	colorDisabled = color.NRGBA{242, 242, 242, 153}
)

// Colors a program sets are composited with normal "source over" alpha
// blending. The window starts opaque in colorBackground, the Window's own
// background is painted over it, and every Container and widget is then
// painted in creation order -- a parent before its children -- so a
// translucent color blends with whatever lies beneath it. The window itself
// is never transparent.

// or returns the program's color, or the default when none was set.
func or(chosen *color.NRGBA, fallback color.RGBA) color.Color {
	if chosen != nil {
		return *chosen
	}
	return fallback
}

// pressedShade darkens a Button background while it is held down: the
// default keeps its v1.8 pressed color, and a program's color is mixed 15%
// toward black.
func pressedShade(chosen *color.NRGBA) color.Color {
	if chosen == nil {
		return colorButtonPress
	}
	c := *chosen
	return color.NRGBA{uint8(uint16(c.R) * 85 / 100), uint8(uint16(c.G) * 85 / 100), uint8(uint16(c.B) * 85 / 100), c.A}
}

var (
	fontOnce   sync.Once
	parsedFont *opentype.Font
	faces      = map[float64]font.Face{}
	facesMu    sync.Mutex
)

// faceAt returns the Go Regular face for one scale factor.
func faceAt(scale float64) font.Face {
	fontOnce.Do(func() {
		parsed, err := opentype.Parse(goregular.TTF)
		if err != nil {
			panic("the embedded Go Regular font does not parse")
		}
		parsedFont = parsed
	})
	facesMu.Lock()
	defer facesMu.Unlock()
	if face, ok := faces[scale]; ok {
		return face
	}
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{Size: fontSize * scale, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic("the embedded Go Regular font has no face")
	}
	faces[scale] = face
	return face
}

// measureScales are the display scales a text must fit at. Hinting makes a
// larger face slightly wider than a scaled small one, so a width measured at
// scale 1 alone clipped the last glyph on a high-density display.
var measureScales = []float64{1, 1.5, 2, 3}

// measureText is a text's width in logical pixels: the widest it renders at
// any supported display scale, so layout is the same on every display and no
// glyph is clipped.
func measureText(text string) int {
	width := 0
	for _, scale := range measureScales {
		width = max(width, int(math.Ceil(float64(font.MeasureString(faceAt(scale), text).Ceil())/scale)))
	}
	return width
}

// frame is what render draws: the model, laid out, and the caret phase.
type frame struct {
	model        *model
	scale        float64
	caretVisible bool
}

// render draws the whole window. The caller holds the model's lock.
func render(f frame) *image.RGBA {
	m, s := f.model, f.scale
	img := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(m.width)*s)), int(math.Ceil(float64(m.height)*s))))
	draw.Draw(img, img.Bounds(), image.NewUniform(colorBackground), image.Point{}, draw.Src)
	if m.background != nil {
		fill(img, img.Bounds(), *m.background)
	}
	m.layout()
	face := faceAt(s)
	for _, id := range m.order {
		drawWidget(img, face, f, m.widgets[id])
	}
	return img
}

func scaled(s float64, x, y, w, h int) image.Rectangle {
	return image.Rect(int(math.Round(float64(x)*s)), int(math.Round(float64(y)*s)),
		int(math.Round(float64(x+w)*s)), int(math.Round(float64(y+h)*s)))
}

func fill(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r.Intersect(img.Bounds()), image.NewUniform(c), image.Point{}, draw.Over)
}

// outline draws a rectangle border of the given thickness in device pixels.
func outline(img *image.RGBA, r image.Rectangle, c color.Color, thickness int) {
	fill(img, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thickness), c)
	fill(img, image.Rect(r.Min.X, r.Max.Y-thickness, r.Max.X, r.Max.Y), c)
	fill(img, image.Rect(r.Min.X, r.Min.Y, r.Min.X+thickness, r.Max.Y), c)
	fill(img, image.Rect(r.Max.X-thickness, r.Min.Y, r.Max.X, r.Max.Y), c)
}

// text draws one line of text starting at device x, vertically centered in
// the device rectangle, clipped to clip.
func text(img *image.RGBA, face font.Face, clip image.Rectangle, x int, box image.Rectangle, value string, c color.Color) {
	target, ok := img.SubImage(clip.Intersect(img.Bounds())).(*image.RGBA)
	if !ok || target.Bounds().Empty() {
		return
	}
	metrics := face.Metrics()
	baseline := box.Min.Y + (box.Dy()+metrics.Ascent.Ceil()-metrics.Descent.Ceil())/2
	drawer := font.Drawer{Dst: target, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, baseline)}
	drawer.DrawString(value)
}

func drawWidget(img *image.RGBA, face font.Face, f frame, w *widget) {
	s, m := f.scale, f.model
	box := scaled(s, w.x, w.y, w.w, w.h)
	line := max(1, int(math.Round(s)))
	focused := m.focus == w.id
	foreground := or(w.foreground, colorText)
	if w.disabled {
		// A disabled widget is drawn as usual and then faded.
		defer fill(img, box, colorDisabled)
	}
	switch w.kind {
	case kindColumn, kindRow:
		if w.background != nil {
			fill(img, box, *w.background)
		}
	case kindLabel:
		if w.background != nil {
			fill(img, box, *w.background)
		}
		text(img, face, box, box.Min.X, box, string(w.text), foreground)
	case kindButton:
		background := or(w.background, colorButton)
		if m.pressed == w.id {
			background = pressedShade(w.background)
		}
		fill(img, box, background)
		border := colorBorder
		if focused {
			border = colorFocus
		}
		outline(img, box, border, line)
		width := font.MeasureString(face, string(w.text)).Ceil()
		text(img, face, box, box.Min.X+(box.Dx()-width)/2, box, string(w.text), foreground)
	case kindTextInput:
		fill(img, box, or(w.background, colorField))
		border := colorBorder
		if focused {
			border = colorFocus
		}
		outline(img, box, border, line)
		inner := box.Inset(int(math.Round(textInputInset * s)))
		inner.Min.Y, inner.Max.Y = box.Min.Y+line, box.Max.Y-line
		if len(w.text) == 0 && !focused {
			text(img, face, inner, inner.Min.X, box, w.placeholder, colorMuted)
			return
		}
		// Keep the caret inside the field: scroll the text horizontally.
		caretX := font.MeasureString(face, string(w.text[:w.caret])).Ceil()
		if caretX-w.scroll > inner.Dx()-line {
			w.scroll = caretX - inner.Dx() + line
		}
		if caretX < w.scroll {
			w.scroll = caretX
		}
		text(img, face, inner, inner.Min.X-w.scroll, box, string(w.text), foreground)
		if focused && f.caretVisible {
			x := inner.Min.X + caretX - w.scroll
			fill(img, image.Rect(x, inner.Min.Y+2*line, x+line, inner.Max.Y-2*line).Intersect(inner), foreground)
		}
	case kindCheckbox:
		if w.background != nil {
			fill(img, box, *w.background)
		}
		size := int(math.Round(checkboxBox * s))
		square := image.Rect(box.Min.X, box.Min.Y+(box.Dy()-size)/2, box.Min.X+size, box.Min.Y+(box.Dy()-size)/2+size)
		if w.checked {
			fill(img, square, colorFocus)
			check(img, square, s)
		} else {
			fill(img, square, colorField)
			border := colorBorder
			if focused {
				border = colorFocus
			}
			outline(img, square, border, line)
		}
		if focused && w.checked {
			outline(img, square.Inset(-2*line), colorFocus, line)
		}
		x := square.Max.X + int(math.Round(checkboxGap*s))
		text(img, face, box, x, box, string(w.text), foreground)
	}
}

// check draws a white check mark inside a filled Checkbox square.
func check(img *image.RGBA, square image.Rectangle, s float64) {
	size := float64(square.Dx())
	at := func(fx, fy float64) (float64, float64) {
		return float64(square.Min.X) + fx*size, float64(square.Min.Y) + fy*size
	}
	x1, y1 := at(0.22, 0.52)
	x2, y2 := at(0.42, 0.72)
	x3, y3 := at(0.78, 0.30)
	width := math.Max(1.5, 2*s)
	segment(img, x1, y1, x2, y2, width, colorField)
	segment(img, x2, y2, x3, y3, width, colorField)
}

// segment paints a thick line segment by distance to the segment.
func segment(img *image.RGBA, x1, y1, x2, y2, width float64, c color.RGBA) {
	half := width / 2
	bounds := image.Rect(int(math.Floor(math.Min(x1, x2)-half)), int(math.Floor(math.Min(y1, y2)-half)),
		int(math.Ceil(math.Max(x1, x2)+half))+1, int(math.Ceil(math.Max(y1, y2)+half))+1).Intersect(img.Bounds())
	dx, dy := x2-x1, y2-y1
	length := dx*dx + dy*dy
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			t := 0.0
			if length > 0 {
				t = math.Max(0, math.Min(1, ((px-x1)*dx+(py-y1)*dy)/length))
			}
			ex, ey := px-(x1+t*dx), py-(y1+t*dy)
			if ex*ex+ey*ey <= half*half {
				img.SetRGBA(x, y, c)
			}
		}
	}
}
