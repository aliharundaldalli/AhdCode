package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// hud draws the viewer's two small overlays with the Go Regular font
// embedded in this helper: the zoom and turn in the bottom-left corner, and
// a one-line hint of the controls for the first seconds.
type hud struct {
	mu    sync.Mutex
	faces map[float64]font.Face
	font  *opentype.Font
}

func newHUD() *hud {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic("the embedded Go Regular font does not parse")
	}
	return &hud{faces: map[float64]font.Face{}, font: parsed}
}

func (h *hud) face(scale float64) font.Face {
	h.mu.Lock()
	defer h.mu.Unlock()
	if face, ok := h.faces[scale]; ok {
		return face
	}
	face, err := opentype.NewFace(h.font, &opentype.FaceOptions{Size: 13 * scale, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic("the embedded Go Regular font has no face")
	}
	h.faces[scale] = face
	return face
}

// label renders one line of white text on a translucent dark pill.
func (h *hud) label(text string, scale float64) *ebiten.Image {
	face := h.face(scale)
	pad := int(math.Round(8 * scale))
	width := font.MeasureString(face, text).Ceil() + 2*pad
	metrics := face.Metrics()
	height := metrics.Ascent.Ceil() + metrics.Descent.Ceil() + pad
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{20, 20, 20, 170}), image.Point{}, draw.Src)
	drawer := font.Drawer{Dst: img, Src: image.NewUniform(color.White), Face: face,
		Dot: fixed.P(pad, pad/2+metrics.Ascent.Ceil())}
	drawer.DrawString(text)
	return ebiten.NewImageFromImage(img)
}

func (v *viewer) drawHUD(screen *ebiten.Image) {
	text := fmt.Sprintf("Zoom: %d%%  ·  Rotation: %d°", v.view.zoomPercent(), v.view.rotationDegrees())
	key := fmt.Sprintf("%s@%g", text, v.scale)
	if key != v.hudLabel {
		if v.hudImage != nil {
			v.hudImage.Deallocate()
		}
		v.hudImage, v.hudLabel = v.hud.label(text, v.scale), key
	}
	margin := 10 * v.scale
	bounds := screen.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(margin, float64(bounds.Dy())-margin-float64(v.hudImage.Bounds().Dy()))
	screen.DrawImage(v.hudImage, op)
	if time.Since(v.started) < hintTime {
		if v.hint == nil {
			v.hint = v.hud.label(hintText, v.scale)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate((float64(bounds.Dx())-float64(v.hint.Bounds().Dx()))/2, margin)
		screen.DrawImage(v.hint, op)
	} else if v.hint != nil {
		v.hint.Deallocate()
		v.hint = nil
	}
}

func colorOf(c [4]float32) color.Color {
	return color.NRGBA{uint8(c[0] * 255), uint8(c[1] * 255), uint8(c[2] * 255), uint8(c[3] * 255)}
}
