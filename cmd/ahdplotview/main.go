// Command ahdplotview is AhdCode's own interactive Plot viewer. Chart.show,
// Figure.show, and Surface.show start it on one view file, and Surface.save
// uses it to render a Surface:
//
//	ahdplotview <view.json>
//
// The view file (see spec.go) is read completely and deleted at once. For a
// Chart or Figure it names a PNG preview rendered by ahdplot, also read and
// deleted, and carries the chart's canonical render request; for a Surface it
// carries the Surface's data. The viewer checks everything, opens its window,
// and then writes exactly one line of JSON to standard output --
// {"ready":true} or {"error":"..."} -- and closes standard output. The
// program that called show() continues as soon as it reads that line; the
// viewer stays open on its own until the user closes it. In render mode it
// writes the Surface's PNG and answers without opening a window.
//
// The viewer only shows the chart. Zooming, panning, turning, and orbiting
// change the view, never the chart, its data, or a file saved later. Its
// toolbar's Save asks the bundled ahdgui helper for a save dialog and, for a
// Chart or Figure, runs the bundled ahdplot renderer on the chart's own
// request, so a saved file is exactly what save() writes; both helpers are
// named by absolute path in the view file. It uses no network, loads no
// plug-in, and runs nothing else.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

// Bounds on everything the viewer accepts.
const (
	maxTitleRunes  = 256
	maxPathBytes   = 4096
	maxPreviewSize = 256 << 20 // bytes of PNG
	maxSide        = 8192      // pixels on each side
	maxPixels      = 40 << 20  // pixels in all
)

// reply is the one handshake line.
type reply struct {
	Ready bool   `json:"ready,omitempty"`
	Error string `json:"error,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		answer(os.Stdout, reply{Error: err.Error()})
		os.Exit(1)
	}
}

// answer writes the handshake line and closes standard output, so a viewer
// that outlives its program never writes to a pipe nobody reads.
func answer(out io.Writer, value reply) {
	encoded, _ := json.Marshal(value)
	_, _ = out.Write(append(encoded, '\n'))
	if file, ok := out.(*os.File); ok {
		_ = file.Close()
	}
}

func run(args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: ahdplotview <view.json>")
	}
	spec, err := readSpec(args[0])
	if err != nil {
		return err
	}
	headless := os.Getenv("AHDCODE_PLOTVIEW_HEADLESS") == "1"
	switch spec.Mode {
	case modeRender:
		return renderSurfaceFile(spec, out)
	case modeSurface:
		if headless {
			return runSurfaceHeadless(spec, out)
		}
		return runSurfaceWindow(spec, out)
	}
	img, err := loadPreview(spec.Preview)
	if err != nil {
		return err
	}
	if headless {
		return runHeadless(img, spec, out)
	}
	return runWindow(img, spec, out)
}

// loadPreview reads the preview into memory, deletes the file -- the viewer
// needs nothing on disk while it is open -- and decodes it within the bounds.
func loadPreview(path string) (image.Image, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("the chart preview could not be read")
	}
	if info.Size() > maxPreviewSize {
		_ = os.Remove(path)
		return nil, errors.New("the chart preview is too large")
	}
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil {
		return nil, errors.New("the chart preview could not be read")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("the chart preview is not a valid PNG image")
	}
	if config.Width < 1 || config.Height < 1 || config.Width > maxSide || config.Height > maxSide || config.Width*config.Height > maxPixels {
		return nil, fmt.Errorf("the chart preview is %dx%d pixels; the viewer shows at most %d on a side", config.Width, config.Height, maxSide)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("the chart preview is not a valid PNG image")
	}
	return img, nil
}

// initialWindow is the window size, in points, for an image: its natural
// size, kept between 480x360 and the given bounds.
func initialWindow(imageW, imageH, boundW, boundH int) (int, int) {
	return min(max(imageW, 480), max(boundW, 480)), min(max(imageH, 360), max(boundH, 360))
}
