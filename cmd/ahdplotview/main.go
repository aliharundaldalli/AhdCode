// Command ahdplotview is AhdCode's own interactive Plot viewer. Chart.show
// and Figure.show render a PNG preview with the ahdplot renderer and start
// this helper on it:
//
//	ahdplotview <preview.png> <title>
//
// The viewer reads the preview completely and deletes it, checks it, opens
// its window, and then writes exactly one line of JSON to standard output --
// {"ready":true} or {"error":"..."} -- and closes standard output. The program
// that called show() continues as soon as it reads that line; the viewer
// stays open on its own until the user closes it.
//
// The viewer only shows the image. Zooming, panning, and turning change the
// view, never the chart, its data, or a file saved later. It uses no
// network, runs no command, loads no plug-in, and reads no file except the
// preview it was given.
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
	"strings"
	"unicode/utf8"
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
	if len(args) != 2 {
		return errors.New("usage: ahdplotview <preview.png> <title>")
	}
	path, title := args[0], args[1]
	if path == "" || len(path) > maxPathBytes || strings.ContainsRune(path, 0) {
		return errors.New("invalid preview path")
	}
	if !utf8.ValidString(title) || utf8.RuneCountInString(title) > maxTitleRunes || strings.ContainsRune(title, 0) {
		return errors.New("invalid window title")
	}
	img, err := loadPreview(path)
	if err != nil {
		return err
	}
	if os.Getenv("AHDCODE_PLOTVIEW_HEADLESS") == "1" {
		return runHeadless(img, out)
	}
	return runWindow(img, title, out)
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
