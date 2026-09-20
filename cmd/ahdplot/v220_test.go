package main

import (
	"bytes"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/plotproto"
)

// The v2.2 renderer: a pie and a heatmap, in every format a Chart saves to.
// The helper reads JSON from a file and trusts none of it, so the rejection
// cases matter as much as the drawings.

func pieSpec() plotproto.ChartSpec {
	return plotproto.ChartSpec{
		Present: true, Kind: "pie", Title: "Year 3", Legend: true,
		PieLabels: []string{"Analysis", "Algebra", "Geometry", "Statistics", "Programming"},
		PieValues: []float64{84, 81, 88, 83, 95},
	}
}

func heatmapSpec() plotproto.ChartSpec {
	return plotproto.ChartSpec{
		Present: true, Kind: "heatmap", Title: "Student performance",
		XLabel: "Academic year", YLabel: "Course", Legend: true,
		HeatmapXLabels: []string{"Year 1", "Year 2", "Year 3"},
		HeatmapYLabels: []string{"Analysis", "Algebra", "Geometry", "Statistics", "Programming"},
		HeatmapValues: [][]float64{
			{72, 78, 84}, {68, 75, 81}, {80, 82, 88}, {65, 74, 83}, {85, 91, 95},
		},
	}
}

func renderOne(t *testing.T, spec plotproto.ChartSpec, path string) {
	t.Helper()
	request := plotproto.Request{
		OutputPath: path, Width: 800, Height: 600, Rows: 1, Columns: 1,
		Charts: []plotproto.ChartSpec{spec},
	}
	if err := render(request); err != nil {
		t.Fatalf("rendering %s: %v", filepath.Ext(path), err)
	}
}

// TestV220RendersEveryFormat checks the file is really the format its
// extension claims, not merely that something was written.
func TestV220RendersEveryFormat(t *testing.T) {
	directory := t.TempDir()
	for name, spec := range map[string]plotproto.ChartSpec{"pie": pieSpec(), "heatmap": heatmapSpec()} {
		for _, extension := range []string{".png", ".svg", ".pdf"} {
			path := filepath.Join(directory, name+extension)
			renderOne(t, spec, path)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s%s: %v", name, extension, err)
			}
			if len(content) == 0 {
				t.Fatalf("%s%s is empty", name, extension)
			}
			switch extension {
			case ".png":
				image, err := png.Decode(bytes.NewReader(content))
				if err != nil {
					t.Fatalf("%s.png does not decode: %v", name, err)
				}
				if image.Bounds().Dx() == 0 || image.Bounds().Dy() == 0 {
					t.Fatalf("%s.png has no pixels", name)
				}
			case ".svg":
				text := string(content)
				if !strings.Contains(text, "<svg") {
					t.Fatalf("%s.svg is not SVG", name)
				}
				// A chart is a drawing, not a document that fetches things.
				for _, forbidden := range []string{"<script", "xlink:href=\"http", "<foreignObject"} {
					if strings.Contains(text, forbidden) {
						t.Fatalf("%s.svg contains %s", name, forbidden)
					}
				}
			case ".pdf":
				if !bytes.HasPrefix(content, []byte("%PDF-")) {
					t.Fatalf("%s.pdf has no PDF header", name)
				}
			}
		}
	}
}

// The same request twice produces the same bytes: no palette is chosen at
// random and no timestamp reaches a PNG.
func TestV220RendersDeterministically(t *testing.T) {
	directory := t.TempDir()
	for name, spec := range map[string]plotproto.ChartSpec{"pie": pieSpec(), "heatmap": heatmapSpec()} {
		first := filepath.Join(directory, name+"-1.png")
		second := filepath.Join(directory, name+"-2.png")
		renderOne(t, spec, first)
		renderOne(t, spec, second)
		one, _ := os.ReadFile(first)
		two, _ := os.ReadFile(second)
		if !bytes.Equal(one, two) {
			t.Fatalf("%s renders differently each time", name)
		}
	}
}

// Every cell of a heatmap is drawn, including the ones in the corners,
// where a coordinate sits on the axis boundary and a cell is easiest to
// lose.
//
// Two grids differing only in one corner cell must render differently. The
// rest of the grid pins the colour range to 0..100, so changing that cell
// between two values inside the range repaints nothing else: if the cell
// were dropped or clipped, the two pictures would be identical.
func TestV220HeatmapDrawsEveryCell(t *testing.T) {
	directory := t.TempDir()
	// corner names which cell varies; every other cell is fixed, and the
	// 0 and 100 among them hold the colour range still.
	for _, corner := range []struct {
		name  string
		row   int
		colum int
	}{
		{"far corner", 2, 2},
		{"origin", 0, 0},
		{"top left", 2, 0},
		{"bottom right", 0, 2},
	} {
		grid := func(value float64) plotproto.ChartSpec {
			values := [][]float64{{0, 40, 60}, {40, 50, 60}, {40, 60, 100}}
			// Keep the range pinned by 0 and 100 wherever the varying cell
			// is not.
			values[corner.row][corner.colum] = value
			if corner.row == 0 && corner.colum == 0 {
				values[1][1] = 0
			}
			if corner.row == 2 && corner.colum == 2 {
				values[1][1] = 100
			}
			return plotproto.ChartSpec{
				Present: true, Kind: "heatmap", Legend: false,
				HeatmapXLabels: []string{"A", "B", "C"},
				HeatmapYLabels: []string{"P", "Q", "R"},
				HeatmapValues:  values,
			}
		}
		first := filepath.Join(directory, corner.name+"-1.png")
		second := filepath.Join(directory, corner.name+"-2.png")
		renderOne(t, grid(20), first)
		renderOne(t, grid(80), second)
		one, err := os.ReadFile(first)
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.ReadFile(second)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(one, two) {
			t.Fatalf("changing the %s cell changed nothing; that cell is not being drawn", corner.name)
		}
	}
}

// A pie whose slices are all equal is still one wedge per category.
func TestV220PieHandlesEqualAndZeroSlices(t *testing.T) {
	directory := t.TempDir()
	for name, spec := range map[string]plotproto.ChartSpec{
		"equal": {Present: true, Kind: "pie", Legend: true,
			PieLabels: []string{"A", "B", "C"}, PieValues: []float64{5, 5, 5}},
		// A zero slice is valid data: it has no wedge, and the legend still
		// names it.
		"zero": {Present: true, Kind: "pie", Legend: true,
			PieLabels: []string{"A", "B", "C"}, PieValues: []float64{0, 5, 0}},
		"single": {Present: true, Kind: "pie", Legend: false,
			PieLabels: []string{"Only"}, PieValues: []float64{1}},
	} {
		renderOne(t, spec, filepath.Join(directory, name+".png"))
	}
}

// Whatever produced the request, the helper checks it again.
func TestV220RejectsMalformedRequests(t *testing.T) {
	directory := t.TempDir()
	long := strings.Repeat("x", plotproto.MaxLabelRunes+1)
	manyLabels := make([]string, plotproto.MaxPieSlices+1)
	manyValues := make([]float64, plotproto.MaxPieSlices+1)
	for index := range manyLabels {
		manyLabels[index], manyValues[index] = "x", 1
	}
	for name, spec := range map[string]plotproto.ChartSpec{
		"pie with no slices":       {Present: true, Kind: "pie"},
		"pie length mismatch":      {Present: true, Kind: "pie", PieLabels: []string{"A", "B"}, PieValues: []float64{1}},
		"pie with a negative":      {Present: true, Kind: "pie", PieLabels: []string{"A"}, PieValues: []float64{-1}},
		"pie of only zeroes":       {Present: true, Kind: "pie", PieLabels: []string{"A", "B"}, PieValues: []float64{0, 0}},
		"pie with NaN":             {Present: true, Kind: "pie", PieLabels: []string{"A"}, PieValues: []float64{math.NaN()}},
		"pie with too many slices": {Present: true, Kind: "pie", PieLabels: manyLabels, PieValues: manyValues},
		"pie with a long label":    {Present: true, Kind: "pie", PieLabels: []string{long}, PieValues: []float64{1}},
		// A pie has no axes, so carrying an axis title is a malformed
		// request rather than something to ignore.
		"pie with an x axis": {Present: true, Kind: "pie", XLabel: "x",
			PieLabels: []string{"A"}, PieValues: []float64{1}},
		"pie with a y axis": {Present: true, Kind: "pie", YLabel: "y",
			PieLabels: []string{"A"}, PieValues: []float64{1}},

		"heatmap with no labels": {Present: true, Kind: "heatmap"},
		"heatmap missing a row": {Present: true, Kind: "heatmap",
			HeatmapXLabels: []string{"A"}, HeatmapYLabels: []string{"P", "Q"},
			HeatmapValues: [][]float64{{1}}},
		"heatmap missing a column": {Present: true, Kind: "heatmap",
			HeatmapXLabels: []string{"A", "B"}, HeatmapYLabels: []string{"P"},
			HeatmapValues: [][]float64{{1}}},
		"heatmap with infinity": {Present: true, Kind: "heatmap",
			HeatmapXLabels: []string{"A"}, HeatmapYLabels: []string{"P"},
			HeatmapValues: [][]float64{{math.Inf(1)}}},
		"heatmap with a long label": {Present: true, Kind: "heatmap",
			HeatmapXLabels: []string{long}, HeatmapYLabels: []string{"P"},
			HeatmapValues: [][]float64{{1}}},
	} {
		request := plotproto.Request{
			OutputPath: filepath.Join(directory, "out.png"), Width: 400, Height: 300,
			Rows: 1, Columns: 1, Charts: []plotproto.ChartSpec{spec},
		}
		if err := render(request); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

// A pie and a heatmap are Charts, so they work as Figure cells beside the
// chart families that came before them.
func TestV220ChartsWorkInsideAFigure(t *testing.T) {
	request := plotproto.Request{
		OutputPath: filepath.Join(t.TempDir(), "figure.png"),
		Width:      1200, Height: 800, Rows: 2, Columns: 2,
		Charts: []plotproto.ChartSpec{
			pieSpec(),
			heatmapSpec(),
			{Present: true, Kind: "bar", BarLabels: []string{"a", "b"}, BarValues: []float64{1, 2}},
			{Present: false, Kind: "empty"},
		},
	}
	if err := render(request); err != nil {
		t.Fatalf("rendering a figure with a pie and a heatmap: %v", err)
	}
}

// Every chart family that existed before v2.2 still renders.
func TestV220LeavesTheOlderChartsAlone(t *testing.T) {
	directory := t.TempDir()
	for name, spec := range map[string]plotproto.ChartSpec{
		"line-scatter": {Present: true, Kind: "line-scatter", Legend: true, Series: []plotproto.SeriesSpec{
			{Kind: "line", Label: "l", X: []float64{1, 2}, Y: []float64{1, 4}},
			{Kind: "scatter", Label: "s", X: []float64{1, 2}, Y: []float64{2, 3}},
		}},
		"bar":       {Present: true, Kind: "bar", BarLabels: []string{"a", "b"}, BarValues: []float64{1, 2}},
		"histogram": {Present: true, Kind: "histogram", HistogramValues: []float64{1, 2, 2, 3}, HistogramBins: 3},
		"box":       {Present: true, Kind: "box", BoxValues: []float64{1, 2, 3, 4}},
		"errorBar": {Present: true, Kind: "errorBar", ErrorX: []float64{1, 2}, ErrorY: []float64{3, 4},
			ErrorLower: []float64{0.5, 0.5}, ErrorUpper: []float64{0.5, 0.5}},
		"empty": {Present: true, Kind: "empty", Title: "nothing yet"},
	} {
		renderOne(t, spec, filepath.Join(directory, name+".png"))
	}
}
