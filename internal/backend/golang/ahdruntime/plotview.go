package ahdruntime

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"
)

// Chart.show and Figure.show open AhdCode's own interactive viewer, the
// bundled ahdplotview helper, on a PNG preview rendered by ahdplot. The
// viewer reads the preview completely, deletes it, opens its window, and
// answers with one line of JSON -- {"ready":true} or {"error":"..."} -- after
// which show() returns while the viewer stays open on its own. Zooming,
// panning, and turning the view never change the Chart or Figure, and never
// change a file saved later. No other application is ever started.

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

// AhdPlotView opens the viewer on a rendered preview and returns once the
// viewer answered. The preview is always removed. It returns a message for a
// PlotError, or "".
func AhdPlotView(preview, title string) string {
	defer os.Remove(preview)
	if !utf8.ValidString(title) || strings.ContainsRune(title, 0) {
		title = ""
	}
	if utf8.RuneCountInString(title) > ahdPlotViewMaxName {
		title = string([]rune(title)[:ahdPlotViewMaxName])
	}
	helper, err := ahdPlotViewDiscover()
	if err != nil {
		return err.Error()
	}
	process := exec.Command(helper, preview, title)
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
	// The viewer stays open on its own; reap it when the user closes it, so
	// a long-running program keeps no zombie process.
	go func() { _ = process.Wait() }()
	return ""
}
