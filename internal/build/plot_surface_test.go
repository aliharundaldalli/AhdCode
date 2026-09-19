package build

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestViewerSaveWritesTheCanonicalChart turns and zooms the view, then saves
// through the viewer's Save path as SVG and PNG: each file is byte for byte
// what chart.save writes, because the viewer runs the renderer on the chart's
// own request.
func TestViewerSaveWritesTheCanonicalChart(t *testing.T) {
	directory := t.TempDir()
	viewPNG, viewSVG, viewPDF := filepath.Join(directory, "view.png"), filepath.Join(directory, "view.svg"), filepath.Join(directory, "view.pdf")
	script, _ := json.Marshal([]map[string]any{{"action": "zoom", "factor": 3, "x": 100, "y": 100}, {"action": "rotateRight"},
		{"action": "save", "path": viewSVG}, {"action": "save", "path": viewPNG}, {"action": "save", "path": viewPDF}, {"action": "close"}})
	report, _ := usePlotViewer(t, string(script))
	source := `bring Plot
x: List<Int> := [1, 2, 3, 4]
chart := Plot.line(x, [1.5, 2.5, 2.0, 4.0]).title("Canonical").xLabel("x").yLabel("y")
chart.save("direct.svg")
chart.save("direct.png")
chart.show()
write("shown")
`
	sources := writeSources(t, map[string]string{"main.ahd": source})
	for _, mode := range []string{"native", "evaluator"} {
		for _, path := range []string{viewPNG, viewSVG, viewPDF} {
			_ = os.Remove(path)
		}
		var out string
		if mode == "native" {
			stdout, stderr, code := buildAndRunIn(t, filepath.Join(sources, "main.ahd"), directory)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			out = stdout
		} else {
			var output, errorOutput bytes.Buffer
			runEvaluatorIn(t, directory, source, &output, &errorOutput)
			out = output.String()
		}
		if out != "shown\n" {
			t.Fatalf("%s output %q", mode, out)
		}
		for _, pair := range [][2]string{{"direct.svg", viewSVG}, {"direct.png", viewPNG}} {
			direct, err1 := os.ReadFile(filepath.Join(directory, pair[0]))
			viewed, err2 := os.ReadFile(pair[1])
			if err1 != nil || err2 != nil || len(direct) == 0 || !bytes.Equal(direct, viewed) {
				t.Fatalf("%s: the viewer's %s differs from chart.save (%v %v)", mode, filepath.Base(pair[1]), err1, err2)
			}
		}
		if pdf, err := os.ReadFile(viewPDF); err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatalf("%s: the viewer's PDF: %v", mode, err)
		}
		states := readReport(t, report)
		if len(states) != 7 || states[2].Rotation != 270 {
			t.Fatalf("%s: states %+v", mode, states)
		}
	}
}

const surfaceProgram = `bring Plot
bring Numeric
from Plot bring (Surface, PlotError)

x: List<Real> := [-1.0, -0.5, 0.0, 0.5, 1.0]
y: List<Int> := [0, 1, 2]
rows: List<List<Real>> := []
for j in y {
    row: Local List<Real> := []
    for i in x {
        row.add(i * i - j * 0.5)
    }
    rows.add(row)
}
surface: Surface := Plot.surface(x, y, Numeric.matrix(rows))
styled := surface.title("Saddle").xLabel("a").yLabel("b").zLabel("height").size(300, 225)
styled.save("surface.png")
styled.wireframe(true).save("wire.png")
styled.show()
attempt {
    styled.save("surface.svg")
} except PlotError as error {
    write(error.message)
}
attempt {
    Plot.surface(x, [0, 1], Numeric.matrix(rows))
} except PlotError as error {
    write(error.message)
}
attempt {
    styled.size(5000, 10)
} except PlotError as error {
    write(error.message)
}
attempt {
    Plot.surface([1, 0], [0, 1], Numeric.matrix([[0.0, 0.0], [0.0, 0.0]]))
} except PlotError as error {
    write(error.message)
}
write("done")
`

const surfaceExpected = `a Surface saves only to .png; use a path ending in .png
the z Matrix has 3 rows; a surface needs one row per y value (2)
surface size must be 1 to 2000 on each side
surface x values must be increasing
done
`

func TestSurfaceSaveAndShowNativeAndEvaluatorAgree(t *testing.T) {
	report, _ := usePlotViewer(t, `[{"action":"orbit","x":45,"y":-20},{"action":"zoom","factor":1.5},{"action":"reset"},{"action":"close"}]`)
	sources := writeSources(t, map[string]string{"main.ahd": surfaceProgram})
	images := map[string][]byte{}
	for _, mode := range []string{"native", "evaluator"} {
		directory := t.TempDir()
		var out string
		if mode == "native" {
			stdout, stderr, code := buildAndRunIn(t, filepath.Join(sources, "main.ahd"), directory)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			out = stdout
		} else {
			var output, errorOutput bytes.Buffer
			runEvaluatorIn(t, directory, surfaceProgram, &output, &errorOutput)
			out = output.String()
		}
		if out != surfaceExpected {
			t.Fatalf("%s output:\n%s", mode, out)
		}
		saved, err := os.ReadFile(filepath.Join(directory, "surface.png"))
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(saved))
		if err != nil || config.Width != 400 || config.Height != 300 {
			t.Fatalf("%s: surface.png %v %v", mode, config, err)
		}
		wire, _ := os.ReadFile(filepath.Join(directory, "wire.png"))
		if bytes.Equal(saved, wire) {
			t.Fatalf("%s: wireframe changed nothing", mode)
		}
		images[mode] = saved
		states := readReport(t, report)
		if len(states) != 5 || states[3].Zoom != 1 {
			t.Fatalf("%s: camera states %+v", mode, states)
		}
		if strings.Contains(out, "PlotError") {
			t.Fatal(out)
		}
	}
	if !bytes.Equal(images["native"], images["evaluator"]) {
		t.Fatal("a native program and the evaluator saved different Surface images")
	}
}
