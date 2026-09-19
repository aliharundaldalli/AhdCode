package build

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

var (
	plotHelpersBuild sync.Once
	plotRenderer     string
	plotViewer       string
	plotHelpersFail  string
)

// usePlotViewer builds the real ahdplot renderer and the real ahdplotview
// viewer once, and runs every viewer in this test headless: it replays script
// and writes its view states to the returned report file. TMPDIR is private
// to the test, so leftover previews can be counted.
func usePlotViewer(t *testing.T, script string) (report, previews string) {
	t.Helper()
	plotHelpersBuild.Do(func() {
		directory, err := os.MkdirTemp("", "ahdplotview-build-test-")
		if err != nil {
			plotHelpersFail = err.Error()
			return
		}
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		plotRenderer = filepath.Join(directory, "ahdplot"+suffix)
		plotViewer = filepath.Join(directory, "ahdplotview"+suffix)
		root, _ := filepath.Abs(filepath.Join("..", ".."))
		for _, build := range []struct{ dir, output, target string }{
			{root, plotRenderer, "./cmd/ahdplot"},
			{filepath.Join(root, "cmd", "ahdplotview"), plotViewer, "."},
		} {
			command := exec.Command("go", "build", "-o", build.output, build.target)
			command.Dir = build.dir
			if output, err := command.CombinedOutput(); err != nil {
				plotHelpersFail = err.Error() + "\n" + string(output)
				return
			}
		}
	})
	if plotHelpersFail != "" {
		t.Fatalf("building the Plot helpers: %s", plotHelpersFail)
	}
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	t.Setenv("AHDCODE_PLOT_RUNTIME", plotRenderer)
	t.Setenv("AHDCODE_PLOTVIEW_RUNTIME", plotViewer)
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS", "1")
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_SCRIPT", script)
	report = filepath.Join(t.TempDir(), "report.json")
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_REPORT", report)
	return report, filepath.Join(temp, "ahdcode", "plot")
}

type viewState struct {
	After    string  `json:"after"`
	ImageW   int     `json:"imageWidth"`
	ImageH   int     `json:"imageHeight"`
	Zoom     float64 `json:"zoom"`
	Rotation int     `json:"rotation"`
}

func readReport(t *testing.T, path string) []viewState {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the viewer wrote no report: %v", err)
	}
	var states []viewState
	if err := json.Unmarshal(data, &states); err != nil {
		t.Fatalf("report %s: %v", data, err)
	}
	return states
}

func leftoverPreviews(t *testing.T, directory string) []string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(directory, "chart-*.png"))
	return matches
}

const viewerScript = `[{"action":"zoom","factor":2,"x":600,"y":300},{"action":"pan","x":-40,"y":25},` +
	`{"action":"rotateRight"},{"action":"rotateRight"},{"action":"rotateLeft"},{"action":"zoom","factor":0.5,"x":10,"y":10},` +
	`{"action":"reset"},{"action":"close"}]`

// showProgram saves the chart, shows it -- the viewer zooms, pans, and
// turns it -- and saves it again: the two files must be identical.
const showProgram = `bring Plot
from Plot bring (Chart, PlotError)

x: List<Int> := [1, 2, 3, 4, 5]
chart := Plot.new()
chart = chart.line(x, [2.0, 4.0, 3.0, 5.0, 7.0], "Line")
chart = chart.scatter(x, [1.5, 3.5, 3.0, 4.5, 6.5], "Points")
chart = chart.title("Viewer").xLabel("x").yLabel("y").legend(true)
chart.save("before.png")
chart.show()
chart.save("after.png")
write("shown and saved")
attempt {
    chart.size(7000, 600).show()
} except PlotError as error {
    write(error.message)
}
`

func TestPlotShowOpensTheFirstPartyViewerAndChangesNothing(t *testing.T) {
	for _, mode := range []string{"native", "evaluator"} {
		t.Run(mode, func(t *testing.T) {
			report, previews := usePlotViewer(t, viewerScript)
			directory := writeSources(t, map[string]string{"main.ahd": showProgram})
			want := "shown and saved\na chart larger than 6144 on a side cannot be shown; save() it instead\n"
			var out string
			if mode == "native" {
				stdout, stderr, code := buildAndRunIn(t, filepath.Join(directory, "main.ahd"), directory)
				if code != 0 {
					t.Fatalf("exit %d: %s", code, stderr)
				}
				out = stdout
			} else {
				var output, errorOutput bytes.Buffer
				runEvaluatorIn(t, directory, showProgram, &output, &errorOutput)
				out = output.String()
			}
			if out != want {
				t.Fatalf("output %q", out)
			}
			before, err1 := os.ReadFile(filepath.Join(directory, "before.png"))
			after, err2 := os.ReadFile(filepath.Join(directory, "after.png"))
			if err1 != nil || err2 != nil || len(before) == 0 || !bytes.Equal(before, after) {
				t.Fatal("save after show differs from save before show")
			}
			states := readReport(t, report)
			// The renderer draws the 800x600 chart at 96 dpi: 1067x800 pixels.
			if len(states) != 9 || states[0].ImageW != 1067 || states[0].ImageH != 800 {
				t.Fatalf("viewer states %+v", states)
			}
			if states[1].Zoom != 2*states[0].Zoom || states[3].Rotation != 270 || states[4].Rotation != 180 || states[5].Rotation != 270 {
				t.Fatalf("viewer transforms %+v", states)
			}
			if states[7].Rotation != 0 || states[7].Zoom != states[0].Zoom {
				t.Fatalf("reset %+v", states[7])
			}
			if left := leftoverPreviews(t, previews); len(left) != 0 {
				t.Fatalf("previews left behind: %v", left)
			}
		})
	}
}

func TestFigureShowUsesTheSameViewer(t *testing.T) {
	// A Figure is 500x400 units per cell, drawn at 96 dpi.
	for _, grid := range []struct {
		rows, columns, charts int
		width, height         int
	}{{1, 1, 1, 667, 533}, {2, 2, 4, 1333, 1067}, {2, 2, 3, 1333, 1067}, {2, 3, 6, 2000, 1067}} {
		report, previews := usePlotViewer(t, `[{"action":"rotateLeft"},{"action":"zoom","factor":3,"x":100,"y":100},{"action":"close"}]`)
		var charts []string
		for index := range grid.charts {
			charts = append(charts, "Plot.bar([\"a\", \"b\"], [1, "+string(rune('2'+index))+"])")
		}
		source := "bring Plot\nfrom Plot bring (Chart, Figure)\n\ncharts: List<Chart> := [" + strings.Join(charts, ", ") + "]\n" +
			"figure := Plot.subplots(" + string(rune('0'+grid.rows)) + ", " + string(rune('0'+grid.columns)) + ", charts)\nfigure.show()\nwrite(\"figure shown\")\n"
		directory := writeSources(t, map[string]string{"main.ahd": source})
		stdout, stderr, code := buildAndRunIn(t, filepath.Join(directory, "main.ahd"), directory)
		if code != 0 || stdout != "figure shown\n" {
			t.Fatalf("%dx%d figure (exit %d, stderr %q): %q", grid.rows, grid.columns, code, stderr, stdout)
		}
		states := readReport(t, report)
		if states[0].ImageW != grid.width || states[0].ImageH != grid.height || states[1].Rotation != 90 {
			t.Fatalf("%dx%d figure viewer states %+v", grid.rows, grid.columns, states)
		}
		if left := leftoverPreviews(t, previews); len(left) != 0 {
			t.Fatalf("previews left behind: %v", left)
		}
	}
}

func TestPlotShowWithoutTheViewerIsAPlotError(t *testing.T) {
	_, previews := usePlotViewer(t, "")
	t.Setenv("AHDCODE_PLOTVIEW_RUNTIME", filepath.Join(t.TempDir(), "missing", "ahdplotview"))
	source := "bring Plot\nfrom Plot bring PlotError\n\nattempt {\n    Plot.line([1, 2], [3, 4]).show()\n} except PlotError as error {\n    write(error.message)\n}\n"
	directory := writeSources(t, map[string]string{"main.ahd": source})
	want := "the Plot viewer (ahdplotview) was not found; reinstall AhdCode with its bundled helpers\n"
	stdout, stderr, code := buildAndRunIn(t, filepath.Join(directory, "main.ahd"), directory)
	if code != 0 || stdout != want {
		t.Fatalf("native (exit %d, stderr %q): %q", code, stderr, stdout)
	}
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, directory, source, &output, &errorOutput)
	if output.String() != want {
		t.Fatalf("evaluator: %q (stderr %q)", output.String(), errorOutput.String())
	}
	if left := leftoverPreviews(t, previews); len(left) != 0 {
		t.Fatalf("a failed show left previews: %v", left)
	}
}

func TestPlotShowReportsAViewerThatCannotOpen(t *testing.T) {
	_, previews := usePlotViewer(t, `[{"action":"explode"}]`)
	source := "bring Plot\nfrom Plot bring PlotError\n\nattempt {\n    Plot.line([1, 2], [3, 4]).show()\n} except PlotError as error {\n    write(error.message)\n}\n"
	directory := writeSources(t, map[string]string{"main.ahd": source})
	stdout, stderr, code := buildAndRunIn(t, filepath.Join(directory, "main.ahd"), directory)
	if code != 0 || stdout != "invalid headless viewer action explode\n" {
		t.Fatalf("exit %d, stderr %q: %q", code, stderr, stdout)
	}
	if left := leftoverPreviews(t, previews); len(left) != 0 {
		t.Fatalf("previews left behind: %v", left)
	}
}

// buildAndRunIn compiles entry and runs it with directory as its working
// directory, so relative save paths land there.
func buildAndRunIn(t *testing.T, entry, directory string) (string, string, int) {
	t.Helper()
	executable, result := BuildProgram(entry, filepath.Join(t.TempDir(), "program"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	return runIn(t, executable, directory)
}

// runEvaluatorIn runs source in the evaluator with directory as its working
// directory.
func runEvaluatorIn(t *testing.T, directory, source string, output, errorOutput *bytes.Buffer) {
	t.Helper()
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": source})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	lowered := lowering.LowerCompilation(frontend)
	if lowered.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
	}
	session := evaluator.New(bufio.NewReader(strings.NewReader("")), output, directory)
	session.ErrorOutput = errorOutput
	if result := session.Execute(lowered.Compilation, 0); result.Failure != nil {
		t.Fatalf("evaluator raised %s: %s", result.Failure.Name, result.Failure.Message)
	}
}
