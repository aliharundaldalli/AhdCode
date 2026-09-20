package ahdruntime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"
)

// Chart.show, Figure.show, and Surface.show open AhdCode's own interactive
// viewer, the bundled ahdplotview helper, on one view file (see
// cmd/ahdplotview/spec.go): for a Chart or Figure, a PNG preview rendered by
// ahdplot and the chart's own render request, and for a Surface, its data.
// The viewer reads the view file and the preview completely, deletes both,
// opens its window, and answers with one line of JSON -- {"ready":true} or
// {"error":"..."} -- after which show() returns while the viewer stays open
// on its own. Zooming, panning, turning, and orbiting never change the chart,
// and never change a file saved later: the viewer's Save button runs ahdplot
// on the chart's own request, or renders the Surface's canonical image,
// exactly as save() does. Surface.save runs the same helper in render mode.
// No other application is ever started.

// AhdPlotViewRuntimeHint is the helper directory recorded when the compiler
// built the program.
var AhdPlotViewRuntimeHint string

const (
	// AhdPlotViewMaxSide is the largest chart side, in size() units, show()
	// accepts: the renderer draws PNG previews at 96 dpi, 4/3 pixels per
	// unit, and the viewer shows at most 8192 pixels on a side. A larger
	// chart can still be saved.
	AhdPlotViewMaxSide = 6144
	ahdPlotViewTimeout = 30 * time.Second
	ahdPlotViewMaxLine = 4096
	ahdPlotViewMaxName = 256
)

// ahdPlotViewDiscover finds the viewer by absolute path: an explicit
// override, the directory recorded at build time, or the installation beside
// the running executable. PATH is never searched.
func ahdPlotViewDiscover() (string, error) {
	name := "ahdplotview"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if AhdPackagedApplication {
		if path, ok := ahdPackagedHelper(name); ok {
			return path, nil
		}
		return "", errors.New("the Plot viewer (ahdplotview) is missing from this application")
	}
	candidates := []string{os.Getenv("AHDCODE_PLOTVIEW_RUNTIME")}
	if AhdPlotViewRuntimeHint != "" {
		candidates = append(candidates, filepath.Join(AhdPlotViewRuntimeHint, name))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(bin, name), filepath.Join(bin, "..", "libexec", "ahdcode", name))
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(filepath.Clean(candidate)); err == nil && info.Mode().IsRegular() {
			return filepath.Abs(filepath.Clean(candidate))
		}
	}
	return "", errors.New("the Plot viewer (ahdplotview) was not found; reinstall AhdCode with its bundled helpers")
}

// AhdPlotShowSizeProblem rejects a chart too large to show before it is
// rendered.
func AhdPlotShowSizeProblem(width, height int64) string {
	if width > AhdPlotViewMaxSide || height > AhdPlotViewMaxSide {
		return "a chart larger than 6144 on a side cannot be shown; save() it instead"
	}
	return ""
}

// ahdPlotViewSpec is the view file's shape, field for field as
// cmd/ahdplotview/spec.go reads it.
type ahdPlotViewSpec struct {
	Version  int                 `json:"version"`
	Mode     string              `json:"mode"`
	Title    string              `json:"title,omitempty"`
	Preview  string              `json:"preview,omitempty"`
	Renderer string              `json:"renderer,omitempty"`
	Dialog   string              `json:"dialog,omitempty"`
	Request  any                 `json:"request,omitempty"`
	Surface  *ahdPlotSurfaceSpec `json:"surface,omitempty"`
	Output   string              `json:"output,omitempty"`
}

func ahdPlotViewTitle(title string) string {
	if !utf8.ValidString(title) || strings.ContainsRune(title, 0) {
		return ""
	}
	if utf8.RuneCountInString(title) > ahdPlotViewMaxName {
		return string([]rune(title)[:ahdPlotViewMaxName])
	}
	return title
}

// ahdPlotViewHelpers are the helpers the viewer's Save runs, by absolute
// path, or "" when one is not installed, which only disables Save.
func ahdPlotViewHelpers() (renderer, dialog string) {
	if path, err := ahdPlotDiscoverRuntime(); err == nil {
		renderer, _ = filepath.Abs(path)
	}
	if path, err := ahdGUIDiscoverRuntime(); err == nil {
		dialog, _ = filepath.Abs(path)
	}
	return renderer, dialog
}

// AhdPlotView opens the viewer on a rendered preview of a Chart or Figure,
// with its render request (whose output path is ignored), and returns once
// the viewer answered. The preview is always removed. It returns a message
// for a PlotError, or "".
func AhdPlotView(preview, title string, request any) string {
	defer os.Remove(preview)
	renderer, dialog := ahdPlotViewHelpers()
	return ahdPlotViewRun(filepath.Dir(preview), ahdPlotViewSpec{Version: 2, Mode: "chart", Title: ahdPlotViewTitle(title),
		Preview: preview, Renderer: renderer, Dialog: dialog, Request: request})
}

// ahdPlotViewRun writes the view file, starts the viewer on it, and waits
// for its answer. The view file is always removed.
func ahdPlotViewRun(directory string, spec ahdPlotViewSpec) string {
	helper, err := ahdPlotViewDiscover()
	if err != nil {
		return err.Error()
	}
	file, err := os.CreateTemp(directory, "view-*.json")
	if err != nil {
		return "could not write the Plot viewer's view file"
	}
	defer os.Remove(file.Name())
	encoded, err := json.Marshal(spec)
	if err == nil {
		_, err = file.Write(encoded)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "could not write the Plot viewer's view file"
	}
	process := exec.Command(helper, file.Name())
	output, err := process.StdoutPipe()
	if err != nil {
		return "could not start the Plot viewer"
	}
	// The viewer's diagnostics are not program output.
	process.Stderr = nil
	if err := process.Start(); err != nil {
		return "could not start the Plot viewer"
	}
	lines := make(chan string, 1)
	go func() {
		reader := bufio.NewReaderSize(output, ahdPlotViewMaxLine)
		line, _ := reader.ReadSlice('\n')
		lines <- string(line)
	}()
	var answer struct {
		Ready bool   `json:"ready"`
		Error string `json:"error"`
	}
	select {
	case line := <-lines:
		if json.Unmarshal([]byte(line), &answer) != nil || (!answer.Ready && answer.Error == "") {
			answer.Error = "the Plot viewer stopped before its window opened"
		}
	case <-time.After(ahdPlotViewTimeout):
		answer.Error = "the Plot viewer did not start in time"
	}
	if !answer.Ready {
		_ = process.Process.Kill()
		_ = process.Wait()
		return answer.Error
	}
	if spec.Mode == "render" {
		// Render mode ends as soon as it answers.
		_ = process.Wait()
		return ""
	}
	// The viewer stays open on its own; reap it when the user closes it, so
	// a long-running program keeps no zombie process.
	go func() { _ = process.Wait() }()
	return ""
}

// ---------------------------------------------------------------------------
// Surface (v2.0)
// ---------------------------------------------------------------------------
//
// Plot.surface(x, y, z) is a small 3D surface: z is a Matrix with one row
// per y value and one column per x value. A Surface is immutable, like a
// Chart: title, labels, size, and wireframe return new Surfaces. It is drawn
// by the Plot viewer's own deterministic software projection, both in the
// window and for Surface.save, which writes PNG only.

// Surface bounds.
const (
	AhdPlotSurfaceMaxGrid = 256
	AhdPlotSurfaceMaxSide = 2000
)

// Pie and heatmap bounds (v2.2). The heatmap's label bound is the Surface
// grid bound on purpose: both are labelled grids, and one number is easier
// to remember than two. They are mirrored in internal/plotproto, which the
// renderer helper checks for itself.
const (
	AhdPlotMaxPieSlices     = 64
	AhdPlotMaxHeatmapLabels = AhdPlotSurfaceMaxGrid
	AhdPlotMaxHeatmapCells  = 65536
)

// AhdSurface is the runtime interchange shape of a Surface value.
type AhdSurface struct {
	X, Y *AhdList[float64]
	Z    *AhdList[*AhdList[float64]]
	// XCategories and YCategories are presentation labels (v2.2): one per x
	// or y coordinate, shown instead of that coordinate's number. Empty
	// means the axis shows its numbers, exactly as it did before v2.2.
	XCategories *AhdList[string]
	YCategories *AhdList[string]
	Title       string
	XLabel      string
	YLabel      string
	ZLabel      string
	Width       int64
	Height      int64
	Wireframe   bool
}

// ahdPlotSurfaceSpec is the Surface in the view file.
type ahdPlotSurfaceSpec struct {
	X           []float64   `json:"x"`
	Y           []float64   `json:"y"`
	Z           [][]float64 `json:"z"`
	XCategories []string    `json:"x_categories,omitempty"`
	YCategories []string    `json:"y_categories,omitempty"`
	Title       string      `json:"title"`
	XLabel      string      `json:"x_label"`
	YLabel      string      `json:"y_label"`
	ZLabel      string      `json:"z_label"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
	Wireframe   bool        `json:"wireframe"`
}

func ahdPlotFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// AhdPlotSurfaceProblem checks a Surface's grid: 2 to 256 finite,
// increasing x and y values, and a finite z Matrix of len(y) rows and len(x)
// columns.
func AhdPlotSurfaceProblem(x, y []float64, z [][]float64) string {
	if len(x) < 2 || len(y) < 2 || len(x) > AhdPlotSurfaceMaxGrid || len(y) > AhdPlotSurfaceMaxGrid {
		return fmt.Sprintf("a surface needs 2 to %d x values and 2 to %d y values; got %d and %d",
			AhdPlotSurfaceMaxGrid, AhdPlotSurfaceMaxGrid, len(x), len(y))
	}
	for name, values := range map[string][]float64{"x": x, "y": y} {
		for index, value := range values {
			if !ahdPlotFinite(value) {
				return "surface " + name + " values must be finite"
			}
			if index > 0 && value <= values[index-1] {
				return "surface " + name + " values must be increasing"
			}
		}
	}
	if len(z) != len(y) {
		return fmt.Sprintf("the z Matrix has %d rows; a surface needs one row per y value (%d)", len(z), len(y))
	}
	for _, row := range z {
		if len(row) != len(x) {
			return fmt.Sprintf("the z Matrix has %d columns; a surface needs one column per x value (%d)", len(row), len(x))
		}
		for _, value := range row {
			if !ahdPlotFinite(value) {
				return "surface z values must be finite; NaN and Infinity cannot be drawn"
			}
		}
	}
	return ""
}

// AhdPlotSurfaceSizeProblem checks Surface.size.
func AhdPlotSurfaceSizeProblem(width, height int64) string {
	if width < 1 || height < 1 || width > AhdPlotSurfaceMaxSide || height > AhdPlotSurfaceMaxSide {
		return fmt.Sprintf("surface size must be 1 to %d on each side", AhdPlotSurfaceMaxSide)
	}
	return ""
}

func ahdPlotRealRows(rows *AhdList[*AhdList[float64]]) [][]float64 {
	if rows == nil {
		return nil
	}
	items := rows.Snapshot()
	result := make([][]float64, len(items))
	for index, row := range items {
		if row != nil {
			result[index] = row.Snapshot()
		}
	}
	return result
}

// AhdPlotSurface is Plot.surface.
func AhdPlotSurface(class *AhdClass, x, y *AhdList[float64], z AhdMatrix) AhdSurface {
	xs, ys, zs := ahdPlotFloats(x), ahdPlotFloats(y), ahdPlotRealRows(z.Rows)
	if problem := AhdPlotSurfaceProblem(xs, ys, zs); problem != "" {
		AhdRaiseClass(class, problem)
	}
	rows := make([]*AhdList[float64], len(zs))
	for index, row := range zs {
		rows[index] = AhdNewList(row...)
	}
	return AhdSurface{X: AhdNewList(xs...), Y: AhdNewList(ys...), Z: AhdNewList(rows...),
		XLabel: "x", YLabel: "y", ZLabel: "z", Width: 800, Height: 600}
}

func AhdPlotSurfaceTitle(s AhdSurface, text string) AhdSurface  { s.Title = text; return s }
func AhdPlotSurfaceXLabel(s AhdSurface, text string) AhdSurface { s.XLabel = text; return s }
func AhdPlotSurfaceYLabel(s AhdSurface, text string) AhdSurface { s.YLabel = text; return s }
func AhdPlotSurfaceZLabel(s AhdSurface, text string) AhdSurface { s.ZLabel = text; return s }
func AhdPlotSurfaceWireframe(s AhdSurface, enabled bool) AhdSurface {
	s.Wireframe = enabled
	return s
}

func AhdPlotSurfaceSize(class *AhdClass, s AhdSurface, width, height int64) AhdSurface {
	if problem := AhdPlotSurfaceSizeProblem(width, height); problem != "" {
		AhdRaiseClass(class, problem)
	}
	s.Width, s.Height = width, height
	return s
}

func ahdPlotSurfaceSpecOf(s AhdSurface) *ahdPlotSurfaceSpec {
	return &ahdPlotSurfaceSpec{X: ahdPlotFloats(s.X), Y: ahdPlotFloats(s.Y), Z: ahdPlotRealRows(s.Z),
		XCategories: ahdPlotStrings(s.XCategories), YCategories: ahdPlotStrings(s.YCategories),
		Title: s.Title, XLabel: s.XLabel, YLabel: s.YLabel, ZLabel: s.ZLabel,
		Width: int(s.Width), Height: int(s.Height), Wireframe: s.Wireframe}
}

// AhdPlotSurfaceXCategories and AhdPlotSurfaceYCategories set the labels
// shown beside each x or y coordinate. They are presentation only: the
// geometry, the Matrix, and the axis titles are untouched, and a Surface
// stays immutable, so each returns a new one.
func AhdPlotSurfaceXCategories(class *AhdClass, s AhdSurface, labels *AhdList[string]) AhdSurface {
	s.XCategories = ahdPlotCategories(class, labels, ahdPlotFloats(s.X), "x")
	return s
}

func AhdPlotSurfaceYCategories(class *AhdClass, s AhdSurface, labels *AhdList[string]) AhdSurface {
	s.YCategories = ahdPlotCategories(class, labels, ahdPlotFloats(s.Y), "y")
	return s
}

// ahdPlotCategories checks one category list against the coordinates it
// labels. There must be exactly one label per coordinate: a list that does
// not line up would silently mislabel the axis.
func ahdPlotCategories(class *AhdClass, labels *AhdList[string], coordinates []float64, axis string) *AhdList[string] {
	texts := ahdPlotStrings(labels)
	if len(texts) != len(coordinates) {
		AhdRaiseClass(class, fmt.Sprintf("%sCategories needs one label per %s value; got %d labels for %d values",
			axis, axis, len(texts), len(coordinates)))
	}
	for _, text := range texts {
		if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
			AhdRaiseClass(class, "a category label must be text without a NUL byte")
		}
		if utf8.RuneCountInString(text) > ahdPlotViewMaxName {
			AhdRaiseClass(class, fmt.Sprintf("a category label is at most %d characters", ahdPlotViewMaxName))
		}
	}
	return AhdNewList(texts...)
}

// AhdPlotSurfaceSaveSpec saves a Surface's canonical PNG through the viewer's
// render mode. It returns a message for a PlotError, or "".
func AhdPlotSurfaceSaveSpec(spec *ahdPlotSurfaceSpec, path, directory string) string {
	if !strings.EqualFold(filepath.Ext(path), ".png") {
		return "a Surface saves only to .png; use a path ending in .png"
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "invalid save path"
	}
	return ahdPlotViewRun(directory, ahdPlotViewSpec{Version: 2, Mode: "render", Surface: spec, Output: absolute})
}

// AhdPlotSurfaceShowSpec opens a Surface in the viewer.
func AhdPlotSurfaceShowSpec(spec *ahdPlotSurfaceSpec, directory string) string {
	_, dialog := ahdPlotViewHelpers()
	return ahdPlotViewRun(directory, ahdPlotViewSpec{Version: 2, Mode: "surface", Title: ahdPlotViewTitle(spec.Title),
		Dialog: dialog, Surface: spec})
}

func AhdPlotSurfaceSave(class *AhdClass, s AhdSurface, path string) {
	if problem := AhdPlotSurfaceSaveSpec(ahdPlotSurfaceSpecOf(s), path, ahdPlotTempDir(class)); problem != "" {
		AhdRaiseClass(class, problem)
	}
}

func AhdPlotSurfaceShow(class *AhdClass, s AhdSurface) {
	if problem := AhdPlotSurfaceShowSpec(ahdPlotSurfaceSpecOf(s), ahdPlotTempDir(class)); problem != "" {
		AhdRaiseClass(class, problem)
	}
}

// AhdPlotSurfaceData builds a Surface's view data from plain values, for the
// evaluator.
func AhdPlotSurfaceData(x, y []float64, z [][]float64, xCategories, yCategories []string, title, xLabel, yLabel, zLabel string, width, height int64, wireframe bool) *ahdPlotSurfaceSpec {
	return &ahdPlotSurfaceSpec{X: x, Y: y, Z: z,
		XCategories: xCategories, YCategories: yCategories,
		Title: title, XLabel: xLabel, YLabel: yLabel, ZLabel: zLabel,
		Width: int(width), Height: int(height), Wireframe: wireframe}
}
