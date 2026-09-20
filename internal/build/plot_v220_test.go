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

// v2.2's three additions, end to end: the same program through the
// evaluator and a native executable, the files they write, and the viewer's
// canonical Save.

const v220Program = `bring Plot
bring Numeric
from Plot bring (Chart, Surface, PlotError)
from Numeric bring Matrix

courses: List<String> := ["Analysis", "Algebra", "Geometry", "Statistics", "Programming"]
years: List<String> := ["Year 1", "Year 2", "Year 3"]
grades: Matrix := Numeric.matrix([
    [72.0, 78.0, 84.0]
    [68.0, 75.0, 81.0]
    [80.0, 82.0, 88.0]
    [65.0, 74.0, 83.0]
    [85.0, 91.0, 95.0]
])

pie: Chart := Plot.pie(courses, [84, 81, 88, 83, 95])
pie = pie.title("Year 3 course distribution")
pie.save("pie.png")
pie.save("pie.svg")
pie.save("pie.pdf")

heat: Chart := Plot.heatmap(years, courses, grades)
heat = heat.title("Student performance").xLabel("Academic year").yLabel("Course")
heat.save("heat.png")
heat.save("heat.svg")
heat.save("heat.pdf")

plain: Chart := heat.legend(false)
plain.save("heat-no-legend.png")

surface: Surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], grades)
named: Surface := surface.xCategories(years).yCategories(courses)
named = named.xLabel("Academic year").yLabel("Course").zLabel("Grade")
named.save("surface.png")
surface.save("surface-plain.png")

write("saved")
`

// TestV220ChartsAreIdenticalInTheEvaluatorAndNatively is the parity check:
// the same program, run both ways, must write the same bytes.
func TestV220ChartsAreIdenticalInTheEvaluatorAndNatively(t *testing.T) {
	files := []string{
		"pie.png", "pie.svg", "pie.pdf",
		"heat.png", "heat.svg", "heat.pdf",
		"heat-no-legend.png", "surface.png", "surface-plain.png",
	}
	usePlotHelpers(t)
	sources := writeSources(t, map[string]string{"main.ahd": v220Program})
	written := map[string]map[string][]byte{}
	for _, mode := range []string{"evaluator", "native"} {
		directory := t.TempDir()
		var out string
		if mode == "native" {
			stdout, stderr, code := buildAndRunIn(t, filepath.Join(sources, "main.ahd"), directory)
			if code != 0 {
				t.Fatalf("native exit %d: %s", code, stderr)
			}
			out = stdout
		} else {
			var output, errorOutput bytes.Buffer
			runEvaluatorIn(t, directory, v220Program, &output, &errorOutput)
			out = output.String()
		}
		if out != "saved\n" {
			t.Fatalf("%s output %q", mode, out)
		}
		written[mode] = map[string][]byte{}
		for _, name := range files {
			content, err := os.ReadFile(filepath.Join(directory, name))
			if err != nil || len(content) == 0 {
				t.Fatalf("%s did not write %s: %v", mode, name, err)
			}
			written[mode][name] = content
		}
	}
	for _, name := range files {
		if filepath.Ext(name) == ".pdf" {
			// A PDF carries renderer metadata that is not byte stable, so
			// the repository compares PNG and SVG byte for byte and checks
			// a PDF for validity, exactly as the viewer Save test does.
			continue
		}
		if !bytes.Equal(written["evaluator"][name], written["native"][name]) {
			t.Fatalf("%s differs between the evaluator and a native build", name)
		}
	}

	// Each file is really what its extension says.
	for _, name := range files {
		content := written["native"][name]
		switch filepath.Ext(name) {
		case ".png":
			if _, err := png.Decode(bytes.NewReader(content)); err != nil {
				t.Fatalf("%s does not decode as PNG: %v", name, err)
			}
		case ".svg":
			if !strings.Contains(string(content), "<svg") {
				t.Fatalf("%s is not SVG", name)
			}
		case ".pdf":
			if !bytes.HasPrefix(content, []byte("%PDF-")) {
				t.Fatalf("%s has no PDF header", name)
			}
		}
	}
	// legend(false) really changes a heatmap: the colour scale is gone.
	if bytes.Equal(written["native"]["heat.png"], written["native"]["heat-no-legend.png"]) {
		t.Fatal("legend(false) left the heatmap unchanged")
	}
	// Naming a Surface's coordinates really changes what is drawn.
	if bytes.Equal(written["native"]["surface.png"], written["native"]["surface-plain.png"]) {
		t.Fatal("xCategories and yCategories left the Surface unchanged")
	}
}

// Every v2.2 value is immutable: deriving one leaves the original alone.
func TestV220ValuesAreImmutable(t *testing.T) {
	const source = `bring Plot
bring Numeric
from Plot bring (Chart, Surface)
from Numeric bring Matrix

courses: List<String> := ["A", "B", "C"]
years: List<String> := ["Y1", "Y2"]
labels: List<String> := ["A", "B", "C"]
values: List<Int> := [1, 2, 3]
grades: Matrix := Numeric.matrix([[1.0, 2.0], [3.0, 4.0], [5.0, 6.0]])

original: Chart := Plot.pie(labels, values)
changed: Chart := original.title("changed")
original.save("pie-original.png")
changed.save("pie-changed.png")

heat: Chart := Plot.heatmap(years, courses, grades)
quiet: Chart := heat.legend(false)
heat.save("heat-original.png")
quiet.save("heat-changed.png")

surface: Surface := Plot.surface([1, 2], [1, 2, 3], grades)
named: Surface := surface.xCategories(years)
surface.save("surface-original.png")
named.save("surface-changed.png")

// The Lists handed to Plot are never touched.
labels.add("D")
values.add(4)
write(str(len(labels)) + " " + str(len(values)))
`
	usePlotHelpers(t)
	directory := t.TempDir()
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, directory, source, &output, &errorOutput)
	if got := output.String(); got != "4 4\n" {
		t.Fatalf("the caller's Lists were disturbed: %q", got)
	}
	for _, pair := range [][2]string{
		{"pie-original.png", "pie-changed.png"},
		{"heat-original.png", "heat-changed.png"},
		{"surface-original.png", "surface-changed.png"},
	} {
		first, err1 := os.ReadFile(filepath.Join(directory, pair[0]))
		second, err2 := os.ReadFile(filepath.Join(directory, pair[1]))
		if err1 != nil || err2 != nil {
			t.Fatalf("%v %v", err1, err2)
		}
		if bytes.Equal(first, second) {
			t.Fatalf("%s and %s are identical; the derived value changed nothing", pair[0], pair[1])
		}
	}
	// Re-saving the original after deriving from it gives the same bytes:
	// the derivation left it alone.
	const again = `bring Plot
from Plot bring Chart
original: Chart := Plot.pie(["A", "B"], [1, 2])
original.save("first.png")
derived: Chart := original.title("t").legend(false).size(400, 300)
derived.save("derived.png")
original.save("second.png")
write("done")
`
	second := t.TempDir()
	output.Reset()
	errorOutput.Reset()
	runEvaluatorIn(t, second, again, &output, &errorOutput)
	first, _ := os.ReadFile(filepath.Join(second, "first.png"))
	last, _ := os.ReadFile(filepath.Join(second, "second.png"))
	if len(first) == 0 || !bytes.Equal(first, last) {
		t.Fatal("deriving from a Chart changed the original")
	}
}

// The domain rules a program can break at run time, in both execution
// paths: each one is a PlotError with a message, never a crash.
func TestV220DomainErrors(t *testing.T) {
	cases := map[string]string{
		"pie length mismatch":     `Plot.pie(["A", "B"], [1])`,
		"empty pie":               `Plot.pie(noLabels, noValues)`,
		"negative pie value":      `Plot.pie(["A"], [-1])`,
		"all-zero pie":            `Plot.pie(["A", "B"], [0, 0])`,
		"pie xLabel":              `Plot.pie(["A"], [1]).xLabel("x")`,
		"pie yLabel":              `Plot.pie(["A"], [1]).yLabel("y")`,
		"empty heatmap labels":    `Plot.heatmap(noLabels, ["A"], Numeric.matrix([[1.0]]))`,
		"heatmap row mismatch":    `Plot.heatmap(["A"], ["P", "Q"], Numeric.matrix([[1.0]]))`,
		"heatmap column mismatch": `Plot.heatmap(["A", "B"], ["P"], Numeric.matrix([[1.0]]))`,
		"too few x categories":    `Plot.surface([1, 2, 3], [1, 2], Numeric.matrix([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])).xCategories(["only one"])`,
		"too many y categories":   `Plot.surface([1, 2, 3], [1, 2], Numeric.matrix([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])).yCategories(["a", "b", "c"])`,
	}
	usePlotHelpers(t)
	for name, expression := range cases {
		source := "bring Plot\nbring Numeric\nfrom Plot bring PlotError\n" +
			"noLabels: List<String> := []\nnoValues: List<Int> := []\n" +
			"attempt {\n    " + expression +
			"\n    write(\"no error\")\n} except PlotError as error {\n    write(\"PlotError\")\n}\n"
		directory := t.TempDir()
		var output, errorOutput bytes.Buffer
		runEvaluatorIn(t, directory, source, &output, &errorOutput)
		if got := strings.TrimSpace(output.String()); got != "PlotError" {
			t.Fatalf("%s: evaluator said %q", name, got)
		}
		sources := writeSources(t, map[string]string{"main.ahd": source})
		stdout, stderr, code := buildAndRunIn(t, filepath.Join(sources, "main.ahd"), directory)
		if code != 0 {
			t.Fatalf("%s: native exit %d: %s", name, code, stderr)
		}
		if got := strings.TrimSpace(stdout); got != "PlotError" {
			t.Fatalf("%s: native said %q", name, got)
		}
	}
}

// The viewer's Save is canonical for a pie and a heatmap too: turning and
// zooming the view plays no part in the file it writes.
func TestV220ViewerSaveWritesTheCanonicalChart(t *testing.T) {
	for _, chart := range []struct{ name, expression string }{
		{"pie", `Plot.pie(["Analysis", "Algebra", "Geometry"], [84, 81, 88]).title("Courses")`},
		{"heatmap", `Plot.heatmap(["Y1", "Y2"], ["A", "B"], Numeric.matrix([[1.0, 2.0], [3.0, 4.0]])).title("Grades")`},
	} {
		directory := t.TempDir()
		viewPNG := filepath.Join(directory, "view.png")
		viewSVG := filepath.Join(directory, "view.svg")
		viewPDF := filepath.Join(directory, "view.pdf")
		script, _ := json.Marshal([]map[string]any{
			{"action": "zoom", "factor": 3, "x": 100, "y": 100},
			{"action": "rotateRight"},
			{"action": "save", "path": viewSVG},
			{"action": "save", "path": viewPNG},
			{"action": "save", "path": viewPDF},
			{"action": "close"},
		})
		usePlotViewer(t, string(script))
		source := "bring Plot\nbring Numeric\nchart := " + chart.expression + "\n" +
			"chart.save(\"direct.svg\")\nchart.save(\"direct.png\")\nchart.show()\nwrite(\"shown\")\n"
		var output, errorOutput bytes.Buffer
		runEvaluatorIn(t, directory, source, &output, &errorOutput)
		if output.String() != "shown\n" {
			t.Fatalf("%s: output %q (%s)", chart.name, output.String(), errorOutput.String())
		}
		for _, pair := range [][2]string{{"direct.svg", viewSVG}, {"direct.png", viewPNG}} {
			direct, err1 := os.ReadFile(filepath.Join(directory, pair[0]))
			viewed, err2 := os.ReadFile(pair[1])
			if err1 != nil || err2 != nil || len(direct) == 0 || !bytes.Equal(direct, viewed) {
				t.Fatalf("%s: the viewer's %s differs from save (%v %v)",
					chart.name, filepath.Base(pair[1]), err1, err2)
			}
		}
		if pdf, err := os.ReadFile(viewPDF); err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatalf("%s: the viewer's PDF: %v", chart.name, err)
		}
	}
}

// usePlotHelpers builds the real ahdplot renderer once and points the
// runtime at it, the same way usePlotViewer does for a viewer test. A chart
// test needs the renderer but no window.
func usePlotHelpers(t *testing.T) {
	t.Helper()
	usePlotViewer(t, `[{"action":"close"}]`)
}
