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
// object per line in each direction: the runtime writes a request to the
// helper's standard input and reads exactly one response line from its
// standard output. Standard error is never part of the protocol. The shape is
// duplicated field for field in internal/backend/golang/ahdruntime/graphics.go,
// which cannot import this module.
const (
	protocolVersion = 1

	// maxRequestBytes bounds one request line. Every legitimate request is a
	// few hundred bytes; a save path is the longest field.
	maxRequestBytes = 16 << 10
	maxDimension    = 4096
	maxTitleRunes   = 256
	maxPathBytes    = 4096
	// maxCommands bounds the drawing commands kept since the last clear, so a
	// runaway loop cannot exhaust memory. Each command is well under 100 bytes.
	maxCommands = 1_000_000
)

// rgba is a validated, non-premultiplied color. It travels as a JSON array of
// four integers; color names and hex text are parsed once, by the runtime.
type rgba [4]uint8

type request struct {
	Op string `json:"op"`

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
}

type response struct {
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
