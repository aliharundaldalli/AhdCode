package lsp

import (
	"strings"
	"testing"
)

const plotV240LSPSource = `bring Plot
from Plot bring (Chart, LineStyle, Marker, LegendPosition)
x: List<Real> := [1.0, 2.0, 3.0]
y: List<Real> := [3.0, 2.0, 1.0]
chart: Chart := Plot.line(x, y)
chart = chart.lineStyle(LineStyle.dashed).lineWidth(2.0).marker(Marker.diamond).markerSize(6.0)
chart = chart.legendPosition(LegendPosition.topLeft)
`

func TestCompletionOffersThePlotV240StylesOverTheWire(t *testing.T) {
	items := completionAt(t, plotV240LSPSource+"chart.", "file:///main.ahd", len(plotV240LSPSource)+len("chart."))
	for _, label := range []string{"lineStyle", "lineWidth", "marker", "markerSize", "legendPosition"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Chart offers no %q over the wire; got %#v", label, items)
		}
	}
	items = completionAt(t, "bring Plot\nfrom Plot bring LineStyle\nLineStyle.", "file:///main.ahd", len("bring Plot\nfrom Plot bring LineStyle\nLineStyle."))
	for _, label := range []string{"solid", "dashed", "dotted", "dashDot"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("LineStyle offers no %q over the wire; got %#v", label, items)
		}
	}
	items = completionAt(t, "bring Plot\nfrom Plot bring Marker\nMarker.", "file:///main.ahd", len("bring Plot\nfrom Plot bring Marker\nMarker."))
	for _, label := range []string{"none", "circle", "square", "triangle", "diamond", "cross"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Marker offers no %q over the wire; got %#v", label, items)
		}
	}
	items = completionAt(t, "bring Plot\nfrom Plot bring LegendPosition\nLegendPosition.", "file:///main.ahd", len("bring Plot\nfrom Plot bring LegendPosition\nLegendPosition."))
	for _, label := range []string{"topRight", "topLeft", "bottomRight", "bottomLeft"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("LegendPosition offers no %q over the wire; got %#v", label, items)
		}
	}
}

func TestHoverAndSignatureHelpForPlotV240StylesOverTheWire(t *testing.T) {
	hover := hoverAt(t, plotV240LSPSource, "file:///main.ahd", "chart.lineStyle(", len("chart.")+1)
	if !strings.Contains(hover.Contents.Value, "lineStyle: (style: LineStyle) -> Chart") {
		t.Fatalf("hover = %q", hover.Contents.Value)
	}
	help, found := signatureHelpAt(t, plotV240LSPSource, "file:///main.ahd", "chart.lineStyle(", len("chart.lineStyle("))
	if !found || len(help.Signatures) == 0 || help.Signatures[0].Label != "(style: LineStyle) -> Chart" {
		t.Fatalf("signature help = %+v, found=%v", help, found)
	}
	help, found = signatureHelpAt(t, plotV240LSPSource, "file:///main.ahd", "chart.legendPosition(", len("chart.legendPosition("))
	if !found || len(help.Signatures) == 0 || help.Signatures[0].Label != "(position: LegendPosition) -> Chart" {
		t.Fatalf("legendPosition signature help = %+v, found=%v", help, found)
	}
}
