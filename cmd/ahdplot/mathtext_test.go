package main

import (
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/plotproto"
)

func TestWholeStringMathTextRendersAcrossPlotLabels(t *testing.T) {
	output := filepath.Join(t.TempDir(), "math.png")
	err := render(plotproto.Request{
		OutputPath: output, Width: 640, Height: 480, Rows: 1, Columns: 1,
		Charts: []plotproto.ChartSpec{{Present: true, Kind: "line-scatter", Title: "$f(x)=x^2$", XLabel: "$x$", YLabel: "$y$", Legend: true,
			Series: []plotproto.SeriesSpec{{Kind: "line", Label: "$f", X: []float64{0, 1}, Y: []float64{0, 1}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() < 1000 {
		t.Fatalf("math plot output = %v, size=%d", err, info.Size())
	}
}

func TestInvalidWholeStringMathTextIsAnError(t *testing.T) {
	for _, malformed := range []string{"$\\frac{$", "$x^{$"} {
		err := render(plotproto.Request{
			OutputPath: filepath.Join(t.TempDir(), "invalid.png"), Width: 640, Height: 480, Rows: 1, Columns: 1,
			Charts: []plotproto.ChartSpec{{Present: true, Kind: "empty", Title: malformed}},
		})
		if err == nil {
			t.Fatalf("invalid math text %q rendered without an error", malformed)
		}
	}
}

func TestMathTextAcceptsSurfaceStyleLabels(t *testing.T) {
	output := filepath.Join(t.TempDir(), "surface-label.png")
	if err := renderMath(plotproto.Request{
		Mode: "math", OutputPath: output,
		Math: &plotproto.MathSpec{Formula: "$f(x,y)$", FontSize: 12},
	}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("surface-style math output = %v, size=%d", err, info.Size())
	}
}

func TestMathTextIsWholeStringOptIn(t *testing.T) {
	for _, value := range []string{"Cost $100", "Variance is $\\sigma^2$", "ordinary text"} {
		if _, ok := mathTextFormula(value); ok {
			t.Fatalf("%q was treated as a whole-string math label", value)
		}
	}
	for _, value := range []string{"$x$", "$f(x,y)$", "$x_0$"} {
		if _, ok := mathTextFormula(value); !ok {
			t.Fatalf("%q was not treated as a whole-string math label", value)
		}
	}
}

func TestMathFormulaQAFormatsAndQuality(t *testing.T) {
	formulas := []string{
		"$x^2$",
		"$f(x)=x^2+3x+2$",
		"$\\alpha+\\beta$",
		"$\\sigma^2$",
		"$\\sum_{i=1}^{n} x_i$",
		"$\\int_a^b f(x)\\,dx$",
		"$f(x,y)=x^2+y^2$",
	}

	formats := []string{"png", "svg", "pdf"}
	dir := t.TempDir()

	for _, formula := range formulas {
		// 1. Chart formats: PNG, SVG, PDF
		for _, format := range formats {
			outPath := filepath.Join(dir, "chart."+format)
			err := render(plotproto.Request{
				OutputPath: outPath, Width: 400, Height: 300, Rows: 1, Columns: 1,
				Charts: []plotproto.ChartSpec{{
					Present: true, Kind: "line-scatter", Title: formula, XLabel: "$x$", YLabel: "$y$",
					Series: []plotproto.SeriesSpec{{Kind: "line", Label: formula, X: []float64{0, 1}, Y: []float64{0, 1}}},
				}},
			})
			if err != nil {
				t.Fatalf("render %s for formula %s failed: %v", format, formula, err)
			}
			info, err := os.Stat(outPath)
			if err != nil || info.Size() == 0 {
				t.Fatalf("output %s for %s is empty or missing: %v", format, formula, err)
			}
		}

		// 2. Surface math PNG
		surfaceOut := filepath.Join(dir, "surface_math.png")
		err := renderMath(plotproto.Request{
			Mode: "math", OutputPath: surfaceOut,
			Math: &plotproto.MathSpec{Formula: formula, FontSize: 14},
		})
		if err != nil {
			t.Fatalf("renderMath for %s failed: %v", formula, err)
		}
		info, err := os.Stat(surfaceOut)
		if err != nil || info.Size() == 0 {
			t.Fatalf("surface math output for %s is empty or missing: %v", formula, err)
		}
	}
}
