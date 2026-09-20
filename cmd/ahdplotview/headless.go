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
// before it answers ready. The window is 800x600 points. A "save" action
// saves to its path without a dialog, as the toolbar's Save would after the
// user chose that path. A Surface replays orbit, zoom, pan, reset, save, and
// close on its camera.

const maxHeadlessActions = 256

type action struct {
	Action string  `json:"action"` // zoom, pan, rotateLeft, rotateRight, orbit, reset, save, close
	Factor float64 `json:"factor,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Path   string  `json:"path,omitempty"`
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

	Saved string `json:"saved,omitempty"`
	Error string `json:"error,omitempty"`

	Azimuth   float64 `json:"azimuth,omitempty"`
	Elevation float64 `json:"elevation,omitempty"`
	PanX      float64 `json:"panX,omitempty"`
	PanY      float64 `json:"panY,omitempty"`
}

func headlessScript() ([]action, error) {
	var script []action
	if text := os.Getenv("AHDCODE_PLOTVIEW_HEADLESS_SCRIPT"); text != "" {
		if err := json.Unmarshal([]byte(text), &script); err != nil || len(script) > maxHeadlessActions {
			return nil, errors.New("invalid headless viewer script")
		}
	}
	return script, nil
}

func writeReport(report any) error {
	if path := os.Getenv("AHDCODE_PLOTVIEW_HEADLESS_REPORT"); path != "" {
		encoded, _ := json.Marshal(report)
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			return errors.New("could not write the headless viewer report")
		}
	}
	return nil
}

// runSurfaceHeadless replays a script on a Surface's camera. The view is
// 800x600 points.
func runSurfaceHeadless(spec viewSpec, out io.Writer) error {
	script, err := headlessScript()
	if err != nil {
		return err
	}
	renderer := newSurfaceRendererWithHelper(spec.Renderer)
	if err := renderer.prepare(*spec.Surface, 1); err != nil {
		return err
	}
	c := initialCamera()
	shot := func(after string, closed bool) state {
		return state{After: after, Zoom: c.zoom, Azimuth: c.azimuth, Elevation: c.elevation, PanX: c.panX, PanY: c.panY, Closed: closed}
	}
	report := []state{shot("open", false)}
	for _, step := range script {
		closed, saved, problem := false, "", ""
		switch step.Action {
		case "orbit":
			c.orbit(step.X, step.Y)
		case "zoom":
			c.zoomBy(step.Factor)
		case "pan":
			c.panBy(step.X, step.Y, 800, 600)
		case "reset":
			c = initialCamera()
		case "save":
			if err := saveSurfaceViewWithRenderer(renderer, *spec.Surface, c, 800, 600, 1, step.Path); err != nil {
				problem = err.Error()
			} else {
				saved = step.Path
			}
		case "close":
			closed = true
		default:
			return errors.New("invalid headless viewer action " + step.Action)
		}
		entry := shot(step.Action, closed)
		entry.Saved, entry.Error = saved, problem
		report = append(report, entry)
		if closed {
			break
		}
	}
	if err := writeReport(report); err != nil {
		return err
	}
	answer(out, reply{Ready: true})
	return nil
}

func runHeadless(img image.Image, spec viewSpec, out io.Writer) error {
	script, err := headlessScript()
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	v := newView(bounds.Dx(), bounds.Dy(), 800, 600)
	report := []state{snapshot("open", v, bounds, false)}
	for _, step := range script {
		closed, saved, problem := false, "", ""
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
		case "save":
			if err := saveChart(spec, step.Path); err != nil {
				problem = err.Error()
			} else {
				saved = step.Path
			}
		case "close":
			closed = true
		default:
			return errors.New("invalid headless viewer action " + step.Action)
		}
		entry := snapshot(step.Action, v, bounds, closed)
		entry.Saved, entry.Error = saved, problem
		report = append(report, entry)
		if closed {
			break
		}
	}
	if err := writeReport(report); err != nil {
		return err
	}
	answer(out, reply{Ready: true})
	return nil
}

func snapshot(after string, v *view, bounds image.Rectangle, closed bool) state {
	return state{After: after, ImageW: bounds.Dx(), ImageH: bounds.Dy(), Zoom: v.zoom, CenterX: v.centerX,
		CenterY: v.centerY, Rotation: v.rotationDegrees(), Closed: closed}
}
