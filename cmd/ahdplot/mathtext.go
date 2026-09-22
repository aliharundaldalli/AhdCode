package main

// Plot uses a whole-string math opt-in. Plain labels continue to use Gonum's
// normal text handler; labels wrapped in $...$ are rendered by the shared
// offline Tectonic authority in cmd/ahdplotmath.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strings"
	"sync"

	"ahdcode/cmd/ahdplotmath"
	"ahdcode/internal/plotproto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type mathTextHandler struct {
	plain       text.Handler
	latexRoot   string
	titleShiftX vg.Length
	mu          sync.Mutex
	valid       map[string]error
	order       []string
}

const defaultMathFontSize = 10

func newMathTextHandler() *mathTextHandler {
	plain := plot.DefaultTextHandler
	return &mathTextHandler{
		plain:     plain,
		latexRoot: plotmath.DiscoverLatexRoot(os.Args[0]),
		valid:     make(map[string]error),
	}
}

func (h *mathTextHandler) Cache() *font.Cache { return h.plain.Cache() }

func (h *mathTextHandler) Extents(fnt font.Font) font.Extents { return h.plain.Extents(fnt) }

func (h *mathTextHandler) Lines(value string) []string {
	if _, ok := mathTextFormula(value); ok {
		return []string{value}
	}
	return h.plain.Lines(value)
}

func (h *mathTextHandler) mathFontSize(fnt font.Font) float64 {
	size := float64(fnt.Size)
	if size <= 0 {
		size = defaultMathFontSize
	}
	return size * 1.25
}

func (h *mathTextHandler) Box(value string, fnt font.Font) (vg.Length, vg.Length, vg.Length) {
	formula, ok := mathTextFormula(value)
	if !ok {
		return h.plain.Box(value, fnt)
	}
	asset, err := plotmath.Render(formula, h.mathFontSize(fnt), h.latexRoot)
	if err != nil {
		// validateChartMath normally catches this before Gonum lays out a
		// plot. A late renderer failure must never turn into raw TeX or a
		// Unicode approximation on the canvas.
		return 0, 0, 0
	}
	return h.mathBox(asset, fnt)
}

// mathBox maps the tightly cropped raster onto Gonum's baseline-based text
// contract. The raster has no font-face baseline of its own, so reserve a
// small, stable descender band from the ordinary plot font. This keeps
// superscripts/subscripts from changing the anchor unexpectedly while still
// giving titles, axes, and legend entries comparable vertical rhythm.
func (h *mathTextHandler) mathBox(asset plotmath.Asset, fnt font.Font) (vg.Length, vg.Length, vg.Length) {
	total := vg.Length(asset.Height)
	if total <= 0 {
		return vg.Length(asset.Width), 0, 0
	}
	ext := h.Extents(fnt)
	depth := vg.Length(math.Min(float64(total)*0.25, float64(ext.Descent)+float64(fnt.Size)*0.08))
	if depth < vg.Points(0.5) {
		depth = vg.Points(0.5)
	}
	if depth >= total {
		depth = total * 0.2
	}
	return vg.Length(asset.Width), total - depth, depth
}

func (h *mathTextHandler) Draw(canvas vg.Canvas, value string, style text.Style, point vg.Point) {
	if style.YAlign == draw.YTop && h.titleShiftX != 0 {
		point.X += h.titleShiftX
	}
	formula, ok := mathTextFormula(value)
	if !ok {
		h.plain.Draw(canvas, value, style, point)
		return
	}
	asset, err := plotmath.Render(formula, h.mathFontSize(style.Font), h.latexRoot)
	if err != nil {
		return
	}

	w, ht, depth := h.mathBox(asset, style.Font)

	if style.Rotation != 0 {
		canvas.Push()
		canvas.Rotate(style.Rotation)
	}

	sin64, cos64 := math.Sincos(style.Rotation)
	cos := vg.Length(cos64)
	sin := vg.Length(sin64)
	pt := vg.Point{
		X: point.Y*sin + point.X*cos,
		Y: point.Y*cos - point.X*sin,
	}

	xoffs := vg.Length(style.XAlign) * w
	total := ht + depth
	baseline := pt.Y + vg.Length(style.YAlign)*total - style.FontExtents().Ascent + style.Font.Size

	img := tintMathImage(asset.Image, style.Color)
	if isPDFCanvas(canvas) {
		img = flipVertical(img)
	}

	canvas.DrawImage(vg.Rectangle{
		Min: vg.Point{X: pt.X + xoffs, Y: baseline - depth},
		Max: vg.Point{X: pt.X + xoffs + w, Y: baseline + ht},
	}, img)

	if style.Rotation != 0 {
		canvas.Pop()
	}
}

func isPDFCanvas(c vg.Canvas) bool {
	for c != nil {
		if strings.Contains(fmt.Sprintf("%T", c), "vgpdf") {
			return true
		}
		if dc, ok := c.(*draw.Canvas); ok {
			c = dc.Canvas
			continue
		}
		if dc, ok := c.(draw.Canvas); ok {
			c = dc.Canvas
			continue
		}
		break
	}
	return false
}

func flipVertical(img image.Image) image.Image {
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		srcY := b.Max.Y - 1 - (y - b.Min.Y)
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, srcY))
		}
	}
	return out
}

func mathTextFormula(value string) (string, bool) {
	if len(value) < 2 || !strings.HasPrefix(value, "$") || !strings.HasSuffix(value, "$") {
		return "", false
	}
	return value, true
}

func validateMathText(value, subject string, handler *mathTextHandler) error {
	formula, ok := mathTextFormula(value)
	if !ok {
		return nil
	}
	if strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(formula, "$"), "$")) == "" {
		return fmt.Errorf("%s contains an empty math fragment", subject)
	}
	return handler.validateFormula(formula)
}

func (h *mathTextHandler) validateFormula(formula string) error {
	h.mu.Lock()
	if result, ok := h.valid[formula]; ok {
		h.mu.Unlock()
		return result
	}
	h.mu.Unlock()

	_, err := plotmath.Render(formula, defaultMathFontSize, h.latexRoot)
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.valid[formula]; ok {
		return existing
	}
	if len(h.order) >= 128 {
		delete(h.valid, h.order[0])
		h.order = h.order[1:]
	}
	h.valid[formula], h.order = err, append(h.order, formula)
	return err
}

func tintMathImage(input image.Image, tint color.Color) image.Image {
	if tint == nil {
		tint = color.Black
	}
	r, g, b, _ := color.NRGBAModel.Convert(tint).RGBA()
	result := image.NewNRGBA(input.Bounds())
	for y := input.Bounds().Min.Y; y < input.Bounds().Max.Y; y++ {
		for x := input.Bounds().Min.X; x < input.Bounds().Max.X; x++ {
			_, _, _, alpha := color.NRGBAModel.Convert(input.At(x, y)).RGBA()
			result.SetNRGBA(x, y, color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(alpha >> 8)})
		}
	}
	return result
}

func validateChartMath(spec plotproto.ChartSpec, handler *mathTextHandler) error {
	check := func(value, subject string) error { return validateMathText(value, subject, handler) }
	if err := check(spec.Title, "chart title"); err != nil {
		return err
	}
	if err := check(spec.XLabel, "x label"); err != nil {
		return err
	}
	if err := check(spec.YLabel, "y label"); err != nil {
		return err
	}
	for index, series := range spec.Series {
		if err := check(series.Label, fmt.Sprintf("series %d label", index)); err != nil {
			return err
		}
	}
	labels := append([]string{}, spec.BarLabels...)
	labels = append(labels, spec.PieLabels...)
	labels = append(labels, spec.HeatmapXLabels...)
	labels = append(labels, spec.HeatmapYLabels...)
	for index, value := range labels {
		if err := check(value, fmt.Sprintf("category label %d", index)); err != nil {
			return err
		}
	}
	return nil
}
