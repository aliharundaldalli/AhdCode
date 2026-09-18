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
)

// command is one drawing primitive in Canvas (Cartesian) coordinates. The
// window image, the PNG export, and the SVG export are all derived from the
// same command list, so they cannot disagree about what was drawn.
type command struct {
	kind commandKind
	// line: a,b = start, c,d = end. circle: a,b = center, c = radius.
	// rectangle: a,b = lower-left corner, c,d = width and height.
	a, b, c, d float64
	stroke     rgba
	fill       rgba
	hasFill    bool
	width      float64
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
