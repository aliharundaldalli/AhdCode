package plotmath

import (
	"image"
	"os"
	"strings"
	"testing"
)

func TestValidateAcceptsRequiredMathAndRejectsMalformedOrHostileInput(t *testing.T) {
	for _, formula := range []string{
		"$\\alpha+\\beta$",
		"$\\sigma^2$",
		"$\\sum_{i=1}^{n} x_i$",
		"$\\int_a^b \\frac{1}{2}\\,dx$",
		"$\\sqrt{x}$",
	} {
		if err := validate(formula); err != nil {
			t.Errorf("validate(%q) = %v", formula, err)
		}
	}
	for _, formula := range []string{
		"ordinary text",
		"$",
		"$$",
		"$\\frac{$",
		"$x^{$",
		"$x$y$",
		"$x% comment$",
		"$\\input{/tmp/pwned}$",
		"$\\includegraphics{secret}$",
		"$\\documentclass{article}$",
		"$\\usepackage{shellesc}$",
		"$\\write18{touch /tmp/pwned}$",
		"$\\openout0=/tmp/pwned$",
		"$\\directlua{os.execute('id')}$",
		"$\\expandafter\\def$",
		"$\\message{secret}$",
	} {
		if err := validate(formula); err == nil {
			t.Errorf("validate(%q) accepted unsafe or malformed input", formula)
		}
	}
}

func TestBundleURLEscapesFileNames(t *testing.T) {
	got := bundleURL(`/tmp/Ahd Code/#formula?.ttb`)
	for _, fragment := range []string{"file://", "Ahd%20Code", "%23formula%3F.ttb"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("bundleURL() = %q, missing %q", got, fragment)
		}
	}
}

func TestBoundedLogReportsAllBytesWhileCappingStoredOutput(t *testing.T) {
	log := &boundedLog{}
	input := make([]byte, 64<<10)
	n, err := log.Write(input)
	if err != nil || n != len(input) {
		t.Fatalf("Write() = (%d, %v), want (%d, nil)", n, err, len(input))
	}
	if len(log.data) != 32<<10 {
		t.Fatalf("stored log length = %d, want %d", len(log.data), 32<<10)
	}
}

func TestRenderCachesAFormula(t *testing.T) {
	root := DiscoverLatexRoot(os.Args[0])
	if root == "" {
		t.Skip("bundled offline Tectonic runtime is not present in this source checkout")
	}
	ResetCache()
	first, err := Render("$x^2$", 12, root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render("$x^2$", 12, root)
	if err != nil {
		t.Fatal(err)
	}
	if CompileCount() != 1 {
		t.Fatalf("compiled the same formula %d times", CompileCount())
	}
	if first.Image == nil || second.Image == nil || first.Width <= 0 || first.Height <= 0 {
		t.Fatalf("cached math asset has invalid dimensions: first=%+v second=%+v", first, second)
	}
}

func TestRenderTightCropTransparentBoundingBox(t *testing.T) {
	root := DiscoverLatexRoot(os.Args[0])
	if root == "" {
		t.Skip("bundled offline Tectonic runtime is not present in this source checkout")
	}
	ResetCache()
	asset, err := Render("$x^2$", 12, root)
	if err != nil {
		t.Fatal(err)
	}
	// Regression guard: uncropped asset was > 1000 pt wide and > 300 pt high
	// due to inverting the transparent gopdf canvas into solid black.
	if asset.Width > 50 || asset.Height > 50 {
		t.Fatalf("asset was not tightly cropped: width=%.1f height=%.1f (want < 50)", asset.Width, asset.Height)
	}
	// Background must remain transparent
	bounds := asset.Image.Bounds()
	_, _, _, a := asset.Image.At(bounds.Min.X, bounds.Min.Y).RGBA()
	if a > 0 {
		t.Fatalf("asset top-left corner is opaque (a=%d), expected transparent background", a)
	}
}

func TestTexDocumentKeepsDisplayStyleForMathLabels(t *testing.T) {
	document := texDocument("$\\sum_{i=1}^{n} x_i$")
	if !strings.Contains(document, "\\[\\sum_{i=1}^{n} x_i\\]") {
		t.Fatalf("math wrapper no longer uses display style:\n%s", document)
	}
	if strings.Contains(document, "\\(") || strings.Contains(document, "\\)") {
		t.Fatalf("math wrapper must not switch to inline math:\n%s", document)
	}
}

func TestRenderMandatoryMathUsesEmbeddedFontsAndDisplayGeometry(t *testing.T) {
	root := DiscoverLatexRoot(os.Args[0])
	if root == "" {
		t.Skip("bundled offline Tectonic runtime is not present in this source checkout")
	}
	// A controlled Tectonic run has no need to consult a user font directory.
	// The renderer also supplies no gopdf SubstituteFont callback: a missing
	// embedded glyph is an error, not an excuse to render a host substitute.
	t.Setenv("HOME", t.TempDir())
	ResetCache()
	formulas := []string{
		"$x^2$",
		"$x_i$",
		"$\\alpha+\\beta$",
		"$\\sigma^2$",
		"$\\varepsilon$",
		"$\\sum_{i=1}^{n} x_i$",
		"$\\int_a^b f(x)\\,dx$",
		"$\\frac{a}{b}$",
		"$\\sqrt{x^2+y^2}$",
		"$f(x,y)=x^2+y^2$",
	}
	assets := make(map[string]Asset, len(formulas))
	for _, formula := range formulas {
		asset, err := Render(formula, 12, root)
		if err != nil {
			t.Fatalf("Render(%q) without a user font directory: %v", formula, err)
		}
		if asset.Image == nil || asset.Width <= 0 || asset.Height <= 0 {
			t.Fatalf("Render(%q) produced an invalid asset: %+v", formula, asset)
		}
		if opaqueBounds(asset.Image).Empty() {
			t.Fatalf("Render(%q) produced no ink", formula)
		}
		assets[formula] = asset
	}
	if CompileCount() != uint64(len(formulas)) {
		t.Fatalf("compiled %d formulas, want %d cache misses", CompileCount(), len(formulas))
	}

	baseline := assets["$x^2$"]
	sum := assets["$\\sum_{i=1}^{n} x_i$"]
	integral := assets["$\\int_a^b f(x)\\,dx$"]
	fraction := assets["$\\frac{a}{b}$"]
	squareRoot := assets["$\\sqrt{x^2+y^2}$"]
	for name, asset := range map[string]Asset{
		"display sum":      sum,
		"display integral": integral,
		"fraction":         fraction,
	} {
		if asset.Height <= baseline.Height*1.45 {
			t.Errorf("%s height %.2fpt does not preserve display-style vertical geometry against x^2 at %.2fpt", name, asset.Height, baseline.Height)
		}
	}
	if squareRoot.Width <= baseline.Width {
		t.Errorf("square-root asset width %.2fpt does not cover its radicand against x^2 at %.2fpt", squareRoot.Width, baseline.Width)
	}
	for name, asset := range map[string]Asset{"display sum": sum, "display integral": integral} {
		top, bottom := opaqueInkBands(asset.Image)
		if !top || !bottom {
			t.Errorf("%s lacks ink above and below its operator baseline; display limits may be detached", name)
		}
	}
}

func opaqueBounds(input image.Image) image.Rectangle {
	bounds := input.Bounds()
	minX, minY, maxX, maxY := bounds.Max.X, bounds.Max.Y, bounds.Min.X, bounds.Min.Y
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := input.At(x, y).RGBA()
			if alpha == 0 {
				continue
			}
			minX, minY = minInt(minX, x), minInt(minY, y)
			maxX, maxY = maxInt(maxX, x+1), maxInt(maxY, y+1)
		}
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func opaqueInkBands(input image.Image) (top, bottom bool) {
	bounds := opaqueBounds(input)
	if bounds.Empty() {
		return false, false
	}
	upper := bounds.Min.Y + bounds.Dy()/4
	lower := bounds.Max.Y - bounds.Dy()/4
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := input.At(x, y).RGBA()
			if alpha == 0 {
				continue
			}
			if y < upper {
				top = true
			}
			if y >= lower {
				bottom = true
			}
		}
	}
	return top, bottom
}
