package main

import (
	"encoding/json"
	"errors"
	"image"
	"io"
	"os"
)

// Headless mode is for AhdCode's own automated tests only: with
// AHDCODE_PLOTVIEW_HEADLESS=1 the viewer opens no window, replays the
// actions in AHDCODE_PLOTVIEW_HEADLESS_SCRIPT (a JSON list), and writes the
// view state before and after each action to AHDCODE_PLOTVIEW_HEADLESS_REPORT
// before it answers ready. The window is 800x600 points.

const maxHeadlessActions = 256

type action struct {
	Action string  `json:"action"` // zoom, pan, rotateLeft, rotateRight, reset, close
	Factor float64 `json:"factor,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
}

type state struct {
	After    string  `json:"after"`
	ImageW   int     `json:"imageWidth"`
	ImageH   int     `json:"imageHeight"`
	Zoom     float64 `json:"zoom"`
	CenterX  float64 `json:"centerX"`
	CenterY  float64 `json:"centerY"`
	Rotation int     `json:"rotation"`
	Closed   bool    `json:"closed,omitempty"`
}

func runHeadless(img image.Image, out io.Writer) error {
	var script []action
	if text := os.Getenv("AHDCODE_PLOTVIEW_HEADLESS_SCRIPT"); text != "" {
		if err := json.Unmarshal([]byte(text), &script); err != nil || len(script) > maxHeadlessActions {
			return errors.New("invalid headless viewer script")
		}
	}
	bounds := img.Bounds()
	v := newView(bounds.Dx(), bounds.Dy(), 800, 600)
	report := []state{snapshot("open", v, bounds, false)}
	for _, step := range script {
		closed := false
		switch step.Action {
		case "zoom":
			v.zoomAt(step.Factor, step.X, step.Y)
		case "pan":
			v.pan(step.X, step.Y)
		case "rotateLeft":
			v.rotate(1)
		case "rotateRight":
			v.rotate(-1)
		case "reset":
			v.reset()
		case "close":
			closed = true
		default:
			return errors.New("invalid headless viewer action " + step.Action)
		}
		report = append(report, snapshot(step.Action, v, bounds, closed))
		if closed {
			break
		}
	}
	if path := os.Getenv("AHDCODE_PLOTVIEW_HEADLESS_REPORT"); path != "" {
		encoded, _ := json.Marshal(report)
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			return errors.New("could not write the headless viewer report")
		}
	}
	answer(out, reply{Ready: true})
	return nil
}

func snapshot(after string, v *view, bounds image.Rectangle, closed bool) state {
	return state{After: after, ImageW: bounds.Dx(), ImageH: bounds.Dy(), Zoom: v.zoom, CenterX: v.centerX,
		CenterY: v.centerY, Rotation: v.rotationDegrees(), Closed: closed}
}
