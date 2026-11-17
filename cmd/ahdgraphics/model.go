package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type commandKind uint8

const (
	lineCommand commandKind = iota + 1
	circleCommand
	rectangleCommand
	ellipseCommand
	polygonCommand
	arcCommand
	textCommand
	imageCommand
)

// command is one drawing primitive in Canvas (Cartesian) coordinates. The
// window image, the PNG export, and the SVG export are all derived from the
// same command list, so they cannot disagree about what was drawn.
type command struct {
	kind commandKind
	// line: a,b = start, c,d = end. circle: a,b = center, c = radius.
	// rectangle: a,b = lower-left corner, c,d = width and height.
	// ellipse: a,b = center, c,d = horizontal and vertical radius.
	// arc: a,b = center, c = radius, start and end = angles in degrees.
	// text: a,b = anchor on the baseline, c = font size.
	// image: a,b = lower-left corner, c,d = width and height.
	a, b, c, d float64
	stroke     rgba
	fill       rgba
	hasFill    bool
	width      float64

	start, end float64
	// points holds a polygon's vertices as x0, y0, x1, y1, ...
	points []float64
	text   string
	align  string
	// picture is the decoded file of an image command, shared by every
	// command that draws the same file.
	picture *picture
}

// model is the Canvas state the helper owns: its size, current background,
// and every command drawn since the last clear. generation changes on every
// clear so the window knows to start its image over.
type model struct {
	mu         sync.Mutex
	title      string
	width      int
	height     int
	background rgba
	commands   []command
	generation int
	// held is true while the program is inside a batch: drawing is recorded
	// but the window keeps showing the last complete picture.
	held bool
	// pictures caches decoded image files by path.
	pictures map[string]*picture
}

func newModel(title string, width, height int, background rgba) *model {
	return &model{title: title, width: width, height: height, background: background}
}

// apply validates one drawing request and records it.
func (m *model) apply(value request) error {
	switch value.Op {
	case "clear":
		if value.Color == nil {
			return errors.New("clear requires a color")
		}
		m.mu.Lock()
		m.background = *value.Color
		m.commands = nil
		m.generation++
		m.mu.Unlock()
		return nil
	case "line":
		if value.Color == nil || !finite(value.X1, value.Y1, value.X2, value.Y2, value.LineWidth) || value.LineWidth <= 0 {
			return errors.New("invalid line")
		}
		return m.add(command{kind: lineCommand, a: value.X1, b: value.Y1, c: value.X2, d: value.Y2,
			stroke: *value.Color, width: value.LineWidth})
	case "circle":
		if value.Stroke == nil || !finite(value.X, value.Y, value.Radius, value.LineWidth) ||
			value.Radius < 0 || value.LineWidth <= 0 {
			return errors.New("invalid circle")
		}
		shape := command{kind: circleCommand, a: value.X, b: value.Y, c: value.Radius,
			stroke: *value.Stroke, width: value.LineWidth}
		if value.Fill != nil {
			shape.fill, shape.hasFill = *value.Fill, true
		}
		return m.add(shape)
	case "rectangle":
		if value.Stroke == nil || !finite(value.X, value.Y, value.W, value.H, value.LineWidth) ||
			value.W < 0 || value.H < 0 || value.LineWidth <= 0 {
			return errors.New("invalid rectangle")
		}
		shape := command{kind: rectangleCommand, a: value.X, b: value.Y, c: value.W, d: value.H,
			stroke: *value.Stroke, width: value.LineWidth}
		if value.Fill != nil {
			shape.fill, shape.hasFill = *value.Fill, true
		}
		return m.add(shape)
	case "ellipse":
		if value.Stroke == nil || !finite(value.X, value.Y, value.RX, value.RY, value.LineWidth) ||
			value.RX < 0 || value.RY < 0 || value.LineWidth <= 0 {
			return errors.New("invalid ellipse")
		}
		shape := command{kind: ellipseCommand, a: value.X, b: value.Y, c: value.RX, d: value.RY,
			stroke: *value.Stroke, width: value.LineWidth}
		if value.Fill != nil {
			shape.fill, shape.hasFill = *value.Fill, true
		}
		return m.add(shape)
	case "polygon":
		if value.Stroke == nil || !finite(value.LineWidth) || value.LineWidth <= 0 || !finite(value.Points...) ||
			len(value.Points) < 2*minPolygonPoints || len(value.Points)%2 != 0 || len(value.Points) > 2*maxPolygonPoints {
			return errors.New("invalid polygon")
		}
		shape := command{kind: polygonCommand, points: append([]float64(nil), value.Points...),
			stroke: *value.Stroke, width: value.LineWidth}
		if value.Fill != nil {
			shape.fill, shape.hasFill = *value.Fill, true
		}
		return m.add(shape)
	case "arc":
		if value.Color == nil || !finite(value.X, value.Y, value.Radius, value.Start, value.End, value.LineWidth) ||
			value.Radius < 0 || value.LineWidth <= 0 {
			return errors.New("invalid arc")
		}
		return m.add(command{kind: arcCommand, a: value.X, b: value.Y, c: value.Radius,
			start: value.Start, end: value.End, stroke: *value.Color, width: value.LineWidth})
	case "text":
		if value.Color == nil || !finite(value.X, value.Y, value.Size) || value.Size <= 0 || value.Size > maxTextSize ||
			!validText(value.Text) || !validAlign(value.Align) {
			return errors.New("invalid text")
		}
		return m.add(command{kind: textCommand, a: value.X, b: value.Y, c: value.Size,
			text: value.Text, align: value.Align, stroke: *value.Color})
	case "image":
		if !finite(value.X, value.Y, value.W, value.H) || value.W < 0 || value.H < 0 ||
			value.W > maxImageSide || value.H > maxImageSide {
			return errors.New("invalid image")
		}
		loaded, err := m.picture(value.Path)
		if err != nil {
			return err
		}
		width, height := value.W, value.H
		natural := loaded.pixels.Bounds()
		switch {
		case width == 0 && height == 0:
			width, height = float64(natural.Dx()), float64(natural.Dy())
		case width == 0:
			width = height * float64(natural.Dx()) / float64(natural.Dy())
		case height == 0:
			height = width * float64(natural.Dy()) / float64(natural.Dx())
		}
		return m.add(command{kind: imageCommand, a: value.X, b: value.Y, c: width, d: height, picture: loaded})
	case "hold", "release":
		m.mu.Lock()
		m.held = value.Op == "hold"
		m.mu.Unlock()
		return nil
	}
	return fmt.Errorf("unknown request %q", value.Op)
}

func (m *model) add(value command) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.commands) >= maxCommands {
		return fmt.Errorf("the Canvas holds %d drawing commands since its last clear; clear it before drawing more", maxCommands)
	}
	m.commands = append(m.commands, value)
	return nil
}

// snapshot is a consistent copy for an export, taken under the lock so a
// save never sees a half-applied clear.
type snapshot struct {
	title      string
	width      int
	height     int
	background rgba
	commands   []command
}

func (m *model) snapshot() snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return snapshot{title: m.title, width: m.width, height: m.height, background: m.background,
		commands: append([]command(nil), m.commands...)}
}

// exportFormat reads the format from the path's extension, case-insensitively.
func exportFormat(path string) (string, error) {
	if path == "" || len(path) > maxPathBytes || strings.ContainsRune(path, 0) {
		return "", errors.New("invalid save path")
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "png", nil
	case ".svg":
		return "svg", nil
	}
	return "", errors.New("unsupported file extension; use .png or .svg")
}
