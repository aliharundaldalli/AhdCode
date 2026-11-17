package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

// The protocol between AhdCode's Graphics runtime and this helper is one JSON
// object per line in each direction. The runtime writes a request, carrying an
// id, to the helper's standard input; the helper answers each request with
// exactly one response line echoing that id. Besides responses, the helper
// writes event lines ({"event": ...}, never with an id) for window clicks and
// key presses the program listens for, and one "closed" event when the window
// is gone. Standard error is never part of the protocol. The shape is
// duplicated field for field in internal/backend/golang/ahdruntime/graphics.go,
// which cannot import this module.
const (
	protocolVersion = 3

	// maxRequestBytes bounds one request line. Most requests are a few
	// hundred bytes; a polygon with maxPolygonPoints vertices is the longest.
	maxRequestBytes = 128 << 10
	maxDimension    = 4096
	maxTitleRunes   = 256
	maxPathBytes    = 4096
	// maxCommands bounds the drawing commands kept since the last clear, so a
	// runaway loop cannot exhaust memory. Each command is well under 100 bytes.
	maxCommands = 1_000_000
	// maxScriptEvents bounds the scripted events of a headless test Canvas.
	maxScriptEvents = 64
	// A polygon has between 3 and 2000 vertices.
	minPolygonPoints = 3
	maxPolygonPoints = 2000
	// maxTextRunes bounds one line of drawn text, and maxTextSize its font
	// size in Canvas units.
	maxTextRunes = 1000
	maxTextSize  = 2000
	// maxImageSide bounds the drawn width and height of an image.
	maxImageSide = 4 * maxDimension
)

// rgba is a validated, non-premultiplied color. It travels as a JSON array of
// four integers; color names and hex text are parsed once, by the runtime.
type rgba [4]uint8

type request struct {
	Op string `json:"op"`
	ID int64  `json:"id,omitempty"`

	Version    int    `json:"version,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Title      string `json:"title,omitempty"`
	Headless   bool   `json:"headless,omitempty"`
	Background *rgba  `json:"background,omitempty"`

	X1 float64 `json:"x1,omitempty"`
	Y1 float64 `json:"y1,omitempty"`
	X2 float64 `json:"x2,omitempty"`
	Y2 float64 `json:"y2,omitempty"`

	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Radius float64 `json:"radius,omitempty"`
	W      float64 `json:"w,omitempty"`
	H      float64 `json:"h,omitempty"`

	Color     *rgba   `json:"color,omitempty"`
	Stroke    *rgba   `json:"stroke,omitempty"`
	Fill      *rgba   `json:"fill,omitempty"`
	LineWidth float64 `json:"lineWidth,omitempty"`

	Path string `json:"path,omitempty"`

	// The fields of the v2.8 shapes: an ellipse's radii, a polygon's
	// vertices as x0, y0, x1, y1, ..., an arc's angles in degrees, and one
	// line of text with its font size and horizontal alignment.
	RX     float64   `json:"rx,omitempty"`
	RY     float64   `json:"ry,omitempty"`
	Points []float64 `json:"points,omitempty"`
	Start  float64   `json:"start,omitempty"`
	End    float64   `json:"end,omitempty"`
	Text   string    `json:"text,omitempty"`
	Size   float64   `json:"size,omitempty"`
	Align  string    `json:"align,omitempty"`

	// Kind names the event a listen request turns on: "click", "key",
	// "move", or "drag".
	Kind string `json:"kind,omitempty"`
	// Script is the event list a headless Canvas replays when the program
	// waits. It exists for automated tests without a display; a windowed
	// Canvas rejects it.
	Script []event `json:"script,omitempty"`
}

// event is one line the helper writes on its own: a click, pointer move, or
// drag inside the Canvas in Cartesian coordinates, a key press by its
// normalized name, or closed. A headless test Canvas may also script "tick",
// which the runtime answers by running the program's timers once.
type event struct {
	Event string  `json:"event"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	Key   string  `json:"key,omitempty"`
}

type response struct {
	ID     int64  `json:"id,omitempty"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Closed bool   `json:"closed,omitempty"`
	Open   *bool  `json:"open,omitempty"`
}

var errRequestTooLarge = errors.New("request exceeds the protocol size limit")

// readRequest reads and decodes one bounded request line. io.EOF means the
// runtime closed the pipe, which is how a helper learns its program ended.
func readRequest(reader *bufio.Reader) (request, error) {
	var line []byte
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				break
			}
			return request{}, err
		}
		line = append(line, chunk...)
		if len(line) > maxRequestBytes {
			return request{}, errRequestTooLarge
		}
		if !isPrefix {
			break
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(line)))
	decoder.DisallowUnknownFields()
	var value request
	if err := decoder.Decode(&value); err != nil {
		return request{}, fmt.Errorf("malformed request: %w", err)
	}
	return value, nil
}

func writeResponse(writer *bufio.Writer, value response) error {
	return writeLine(writer, value)
}

func writeLine(writer *bufio.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return writer.Flush()
}

// validateOpen checks the first request. The runtime has already validated
// everything; the helper checks again because it trusts no input.
func validateOpen(value request) error {
	if value.Op != "open" {
		return fmt.Errorf("the first request must be open, not %q", value.Op)
	}
	if value.Version != protocolVersion {
		return fmt.Errorf("protocol version %d is not supported (expected %d)", value.Version, protocolVersion)
	}
	if value.Width < 1 || value.Width > maxDimension || value.Height < 1 || value.Height > maxDimension {
		return fmt.Errorf("canvas size %dx%d is outside 1..%d", value.Width, value.Height, maxDimension)
	}
	if !utf8.ValidString(value.Title) || utf8.RuneCountInString(value.Title) > maxTitleRunes || strings.ContainsRune(value.Title, 0) {
		return errors.New("invalid window title")
	}
	if value.Background == nil {
		return errors.New("open requires a background color")
	}
	if len(value.Script) > 0 && !value.Headless {
		return errors.New("only a headless Canvas replays scripted events")
	}
	if len(value.Script) > maxScriptEvents {
		return errors.New("too many scripted events")
	}
	for _, scripted := range value.Script {
		switch {
		case (scripted.Event == "click" || scripted.Event == "move" || scripted.Event == "drag") && finite(scripted.X, scripted.Y):
		case scripted.Event == "tick":
		case scripted.Event == "key" && keyNameKnown(scripted.Key):
		default:
			return errors.New("invalid scripted event")
		}
	}
	return nil
}

func finite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

// validText accepts one line of printable text.
func validText(text string) bool {
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxTextRunes {
		return false
	}
	for _, character := range text {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func validAlign(align string) bool {
	return align == "left" || align == "center" || align == "right"
}
