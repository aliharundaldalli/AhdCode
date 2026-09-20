package lsp

import (
	"strings"
	"testing"
)

// The v2.2 Plot surface over the real LSP wire, so the protocol layer is
// exercised and not only the analysis store behind it.

const plotV220LSPSource = `bring Plot
bring Numeric
from Plot bring (Chart, Surface)
from Numeric bring Matrix

courses: List<String> := ["Analysis", "Algebra"]
years: List<String> := ["Year 1", "Year 2", "Year 3"]
grades: Matrix := Numeric.matrix([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])
surface: Surface := Plot.surface([1, 2, 3], [1, 2], grades)
named: Surface := surface.xCategories(years)
write(str(len(courses)))
`

func TestCompletionOffersThePlotV220SurfaceOverTheWire(t *testing.T) {
	text := "bring Plot\nx := Plot.\n"
	items := completionAt(t, text, "file:///main.ahd", len(text)-1)
	for _, label := range []string{"pie", "heatmap", "surface", "line", "scatter", "bar", "histogram", "box", "errorBar"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("expected %s among Plot completions, got %#v", label, items)
		}
	}
	text = plotV220LSPSource + "surface."
	items = completionAt(t, text, "file:///main.ahd", len(text))
	for _, label := range []string{"xCategories", "yCategories", "xLabel", "yLabel", "zLabel", "wireframe", "size", "title", "save", "show"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Surface. offers no %q over the wire; got %#v", label, items)
		}
	}
}

func TestHoverAndSignatureHelpForPlotV220OverTheWire(t *testing.T) {
	needle := "surface.xCategories("
	hover := hoverAt(t, plotV220LSPSource, "file:///main.ahd", needle, len("surface.")+1)
	if !strings.Contains(hover.Contents.Value, "xCategories: (labels: List<String>) -> Surface") {
		t.Fatalf("hover = %q", hover.Contents.Value)
	}
	help, found := signatureHelpAt(t, plotV220LSPSource, "file:///main.ahd", needle, len(needle))
	if !found || len(help.Signatures) == 0 {
		t.Fatal("no signature help for xCategories")
	}
	if help.Signatures[0].Label != "(labels: List<String>) -> Surface" {
		t.Fatalf("signature help = %q", help.Signatures[0].Label)
	}
}

// The v2.0 Chart, Figure and Surface members must still answer: this is the
// metadata bug v2.0 fixed, rechecked beside the v2.2 surface.
func TestPlotV200MembersStillAnswerOverTheWire(t *testing.T) {
	chart := "bring Plot\nchart := Plot.new()\nchart."
	items := completionAt(t, chart, "file:///main.ahd", len(chart))
	for _, label := range []string{"title", "xLabel", "yLabel", "legend", "size", "line", "scatter", "save", "show"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Chart offers no %q; got %#v", label, items)
		}
	}
	figure := "bring Plot\nfigure := Plot.subplots(1, 1, [Plot.new()])\nfigure."
	items = completionAt(t, figure, "file:///main.ahd", len(figure))
	for _, label := range []string{"save", "show"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Figure offers no %q; got %#v", label, items)
		}
	}
}
