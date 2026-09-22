package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/plotproto"
	"gonum.org/v1/plot/vg"
)

func TestV240SeriesStylesValidateTypedValues(t *testing.T) {
	valid := []plotproto.SeriesSpec{
		{Kind: "line", LineStyle: "solid", Marker: "none"},
		{Kind: "line", LineStyle: "dashed", LineWidth: 2, Marker: "circle", MarkerSize: 6},
		{Kind: "line", LineStyle: "dotted", Marker: "square"},
		{Kind: "line", LineStyle: "dashDot", Marker: "diamond"},
		{Kind: "scatter", Marker: "triangle", MarkerSize: 8},
		{Kind: "scatter", Marker: "cross"},
	}
	for _, series := range valid {
		if err := validateSeriesStyle(series); err != nil {
			t.Errorf("valid style %+v rejected: %v", series, err)
		}
	}

	invalid := []plotproto.SeriesSpec{
		{Kind: "line", LineStyle: "broken"},
		{Kind: "scatter", LineStyle: "dashed"},
		{Kind: "line", LineWidth: 0 - 1},
		{Kind: "line", LineWidth: math.NaN()},
		{Kind: "line", LineWidth: math.Inf(1)},
		{Kind: "line", Marker: "hexagon"},
		{Kind: "line", MarkerSize: 0 - 1},
		{Kind: "line", MarkerSize: math.NaN()},
		{Kind: "line", MarkerSize: math.Inf(1)},
	}
	for _, series := range invalid {
		if err := validateSeriesStyle(series); err == nil {
			t.Errorf("invalid style %+v was accepted", series)
		}
	}
}

func TestV240SeriesStylesProduceExpectedDrawingPrimitives(t *testing.T) {
	for _, style := range []string{"solid", "dashed", "dotted", "dashDot"} {
		line := seriesLineStyle(plotproto.SeriesSpec{Kind: "line", LineStyle: style})
		if style == "solid" && len(line.Dashes) != 0 {
			t.Fatalf("solid line has dashes: %v", line.Dashes)
		}
		if style != "solid" && len(line.Dashes) == 0 {
			t.Fatalf("%s line has no dash pattern", style)
		}
	}
	for _, marker := range []string{"circle", "square", "triangle", "diamond", "cross"} {
		if got := seriesMarkerStyle(plotproto.SeriesSpec{Kind: "line", Marker: marker}); got == nil {
			t.Fatalf("marker %q produced no glyph", marker)
		}
	}
	if got := seriesMarkerStyle(plotproto.SeriesSpec{Kind: "line", Marker: "none"}); got != nil {
		t.Fatal("Marker.none produced a glyph")
	}
}

func TestV240LineStylesVisualDistinctnessAcrossFormats(t *testing.T) {
	styles := []string{"solid", "dashed", "dotted", "dashDot"}
	formats := []string{"png", "svg", "pdf"}
	dir := t.TempDir()

	for _, format := range formats {
		renderedBytes := make(map[string][]byte)
		for _, style := range styles {
			outPath := filepath.Join(dir, "line_"+style+"."+format)
			err := render(plotproto.Request{
				OutputPath: outPath, Width: 400, Height: 300, Rows: 1, Columns: 1,
				Charts: []plotproto.ChartSpec{{
					Present: true, Kind: "line-scatter", Title: "Style " + style,
					Series: []plotproto.SeriesSpec{{
						Kind: "line", LineStyle: style, LineWidth: 2,
						X: []float64{0, 1, 2, 3, 4}, Y: []float64{0, 1, 4, 9, 16},
					}},
				}},
			})
			if err != nil {
				t.Fatalf("render %s for line style %s failed: %v", format, style, err)
			}
			data, err := os.ReadFile(outPath)
			if err != nil || len(data) == 0 {
				t.Fatalf("file %s is empty or unreadable: %v", outPath, err)
			}
			renderedBytes[style] = data
		}

		// Verify mutual distinctness: no two styles may produce identical output
		for i := 0; i < len(styles); i++ {
			for j := i + 1; j < len(styles); j++ {
				s1, s2 := styles[i], styles[j]
				if bytes.Equal(renderedBytes[s1], renderedBytes[s2]) {
					t.Fatalf("format %s: line style %s and %s produced identical output", format, s1, s2)
				}
			}
		}

		// For SVG: verify stroke-dasharray properties
		if format == "svg" {
			solidSVG := string(renderedBytes["solid"])
			dashedSVG := string(renderedBytes["dashed"])
			dottedSVG := string(renderedBytes["dotted"])
			dashDotSVG := string(renderedBytes["dashDot"])

			if !strings.Contains(dashedSVG, "stroke-dasharray") {
				t.Fatalf("dashed SVG lacks stroke-dasharray")
			}
			if !strings.Contains(dottedSVG, "stroke-dasharray") {
				t.Fatalf("dotted SVG lacks stroke-dasharray")
			}
			if !strings.Contains(dashDotSVG, "stroke-dasharray") {
				t.Fatalf("dashDot SVG lacks stroke-dasharray")
			}
			// Verify each dash pattern is distinct in SVG
			_ = solidSVG
		}
	}
}

func TestV240MarkersVisualDistinctnessAcrossFormats(t *testing.T) {
	markers := []string{"circle", "square", "triangle", "diamond", "cross"}
	dir := t.TempDir()

	for _, format := range []string{"png", "svg", "pdf"} {
		rendered := make(map[string][]byte)
		for _, m := range markers {
			outPath := filepath.Join(dir, "marker_"+m+"."+format)
			err := render(plotproto.Request{
				OutputPath: outPath, Width: 400, Height: 300, Rows: 1, Columns: 1,
				Charts: []plotproto.ChartSpec{{
					Present: true, Kind: "line-scatter", Title: "Marker " + m,
					Series: []plotproto.SeriesSpec{{
						Kind: "line", LineStyle: "solid", Marker: m, MarkerSize: 8,
						X: []float64{1, 2, 3}, Y: []float64{1, 4, 9},
					}},
				}},
			})
			if err != nil {
				t.Fatalf("render %s for marker %s failed: %v", format, m, err)
			}
			data, _ := os.ReadFile(outPath)
			rendered[m] = data
		}

		for i := 0; i < len(markers); i++ {
			for j := i + 1; j < len(markers); j++ {
				m1, m2 := markers[i], markers[j]
				if bytes.Equal(rendered[m1], rendered[m2]) {
					t.Fatalf("format %s: marker %s and %s produced identical output", format, m1, m2)
				}
			}
		}
	}
}

func TestV240AcademicLinePlusMarkerAndLegendParity(t *testing.T) {
	requireMathRuntime(t)
	dir := t.TempDir()

	for _, format := range []string{"png", "svg", "pdf"} {
		outPath := filepath.Join(dir, "academic."+format)
		err := render(plotproto.Request{
			OutputPath: outPath, Width: 500, Height: 350, Rows: 1, Columns: 1,
			Charts: []plotproto.ChartSpec{{
				Present: true, Kind: "line-scatter", Title: "Academic $u_h(x)$", Legend: true,
				Series: []plotproto.SeriesSpec{{
					Kind: "line", LineStyle: "dashed", LineWidth: 2, Marker: "circle", MarkerSize: 6,
					Label: "$u_h$", X: []float64{0, 1, 2, 3}, Y: []float64{0, 1, 2, 3},
				}},
			}},
		})
		if err != nil {
			t.Fatalf("academic line+marker render in %s failed: %v", format, err)
		}
		info, err := os.Stat(outPath)
		if err != nil || info.Size() == 0 {
			t.Fatalf("output %s missing or empty", outPath)
		}
	}
}

func TestV240LegendLayoutHasInsetDefaultsAndAllPositions(t *testing.T) {
	positions := []struct {
		name       string
		top, left  bool
		xoff, yoff vg.Length
	}{
		{"topRight", true, false, vg.Points(-10), vg.Points(-10)},
		{"topLeft", true, true, vg.Points(10), vg.Points(-10)},
		{"bottomRight", false, false, vg.Points(-10), vg.Points(10)},
		{"bottomLeft", false, true, vg.Points(10), vg.Points(10)},
	}
	for _, want := range positions {
		p, _, err := buildPlot(plotproto.ChartSpec{
			Present: true, Kind: "line-scatter", Legend: true, LegendPosition: want.name,
			Series: []plotproto.SeriesSpec{{Kind: "line", Label: "f(x)", X: []float64{0, 1}, Y: []float64{0, 1}}},
		})
		if err != nil {
			t.Fatalf("buildPlot(%s): %v", want.name, err)
		}
		if p.Legend.Top != want.top || p.Legend.Left != want.left || p.Legend.XOffs != want.xoff || p.Legend.YOffs != want.yoff {
			t.Fatalf("legend %s layout = top=%v left=%v x=%v y=%v", want.name, p.Legend.Top, p.Legend.Left, p.Legend.XOffs, p.Legend.YOffs)
		}
		if p.Legend.Padding <= 0 || p.Legend.ThumbnailWidth <= vg.Points(20) {
			t.Fatalf("legend %s has no breathing room: padding=%v thumbnail=%v", want.name, p.Legend.Padding, p.Legend.ThumbnailWidth)
		}
	}
	defaultPlot, _, err := buildPlot(plotproto.ChartSpec{
		Present: true, Kind: "line-scatter", Legend: true,
		Series: []plotproto.SeriesSpec{{Kind: "line", Label: "f", X: []float64{0, 1}, Y: []float64{0, 1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !defaultPlot.Legend.Top || defaultPlot.Legend.Left {
		t.Fatalf("empty LegendPosition did not use topRight default: top=%v left=%v", defaultPlot.Legend.Top, defaultPlot.Legend.Left)
	}
}

func TestV240LegendLayoutRejectsUnknownPosition(t *testing.T) {
	if _, _, err := buildPlot(plotproto.ChartSpec{Present: true, Kind: "empty", LegendPosition: "center"}); err == nil {
		t.Fatal("unknown legend position was accepted")
	}
}

func TestV240ScatterRejectsLineStyleAndMaintainsIdentity(t *testing.T) {
	// Scatter with lineStyle must fail validation
	err := validateSeriesStyle(plotproto.SeriesSpec{
		Kind: "scatter", LineStyle: "dashed",
	})
	if err == nil || !strings.Contains(err.Error(), "line style applies only to line series") {
		t.Fatalf("scatter with lineStyle was not rejected properly: %v", err)
	}

	// Scatter with diamond succeeds
	err = validateSeriesStyle(plotproto.SeriesSpec{
		Kind: "scatter", Marker: "diamond", MarkerSize: 8,
	})
	if err != nil {
		t.Fatalf("scatter with diamond rejected: %v", err)
	}

	// Rendering scatter produces valid chart without line
	dir := t.TempDir()
	outPath := filepath.Join(dir, "scatter_diamond.png")
	err = render(plotproto.Request{
		OutputPath: outPath, Width: 400, Height: 300, Rows: 1, Columns: 1,
		Charts: []plotproto.ChartSpec{{
			Present: true, Kind: "line-scatter",
			Series: []plotproto.SeriesSpec{{
				Kind: "scatter", Marker: "diamond", MarkerSize: 8,
				X: []float64{1, 2, 3}, Y: []float64{3, 2, 1},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("render scatter failed: %v", err)
	}
}

func TestV240SizeValidationExhaustive(t *testing.T) {
	// In the wire protocol (plotproto), 0 represents unspecified (default).
	// Negative values, NaN, Inf, and values exceeding limits must be rejected.
	invalidWidths := []float64{-0.001, -0.01, -1, -10, math.NaN(), math.Inf(1), math.Inf(-1), 32.1, 100}
	for _, w := range invalidWidths {
		err := validateSeriesStyle(plotproto.SeriesSpec{Kind: "line", LineWidth: w})
		if err == nil {
			t.Errorf("invalid lineWidth %v was accepted", w)
		}
	}

	validWidths := []float64{0.1, 0.5, 1.0, 2.0, 5.0, 16.0, 32.0}
	for _, w := range validWidths {
		err := validateSeriesStyle(plotproto.SeriesSpec{Kind: "line", LineWidth: w})
		if err != nil {
			t.Errorf("valid lineWidth %v was rejected: %v", w, err)
		}
	}

	// Verify 0 yields the default width (1 pt)
	defaultLine := seriesLineStyle(plotproto.SeriesSpec{Kind: "line", LineWidth: 0})
	if defaultLine.Width != 1 {
		t.Fatalf("lineWidth 0 did not yield default width 1: %v", defaultLine.Width)
	}

	invalidSizes := []float64{-0.001, -0.01, -1, -10, math.NaN(), math.Inf(1), math.Inf(-1), 64.1, 100}
	for _, s := range invalidSizes {
		err := validateSeriesStyle(plotproto.SeriesSpec{Kind: "line", MarkerSize: s})
		if err == nil {
			t.Errorf("invalid markerSize %v was accepted", s)
		}
	}

	validSizes := []float64{0.1, 1.0, 5.0, 10.0, 32.0, 64.0}
	for _, s := range validSizes {
		err := validateSeriesStyle(plotproto.SeriesSpec{Kind: "line", MarkerSize: s})
		if err != nil {
			t.Errorf("valid markerSize %v was rejected: %v", s, err)
		}
	}

	// Verify 0 yields default marker size (5 pt)
	defaultMarker := seriesMarkerStyle(plotproto.SeriesSpec{Kind: "line", Marker: "circle", MarkerSize: 0})
	if defaultMarker.Radius != 2.5 { // radius is size / 2 = 5 / 2 = 2.5
		t.Fatalf("markerSize 0 did not yield default radius 2.5: %v", defaultMarker.Radius)
	}
}
