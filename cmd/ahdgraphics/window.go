package main

import (
	"errors"
	"image"
	"math"
	"sync"

	"ahdidentity"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The window is only a view: it shows the image the rasterizer draws from
// the command list. Nothing here decides what a drawing looks like, and none
// of the window library's own drawing, input, or timing features is used.

// maxDisplayPixels caps the window image on a high-density display, where a
// Canvas is drawn at the display's scale factor so it stays sharp.
const maxDisplayPixels = maxDimension * maxDimension

type window struct {
	session *session

	readyOnce sync.Once
	ready     chan struct{}

	scale      float64
	generation int
	drawn      int
	display    *image.RGBA
	texture    *ebiten.Image
	keys       []ebiten.Key

	pointer pointerTracker
}

// windowTPS is how often the window samples the pointer. 60 matches a
// typical display, so a drag stays one sample per frame. Timers are not
// driven by this tick; they use the runtime clock.
const windowTPS = 60

// pointerSample is one look at the pointer. The gesture decision does not
// use the window library, so tests can script it.
type pointerSample struct {
	x, y                      int
	pressed                   bool
	justPressed, justReleased bool
}

// pointerTracker remembers whether the current press began inside the
// Canvas. A drag that started inside continues outside. A press that started
// outside never becomes a drag.
type pointerTracker struct {
	known    bool
	x, y     int
	dragging bool
}

// event reports a move or a drag since the previous sample. The press that
// starts a gesture is not itself a drag. ok is false when nothing happened
// that a program should hear.
func (t *pointerTracker) event(sample pointerSample, width, height int, scale float64) (kind string, x, y float64, ok bool) {
	if scale <= 0 {
		scale = 1
	}
	moved := t.known && (sample.x != t.x || sample.y != t.y)
	t.known, t.x, t.y = true, sample.x, sample.y
	if sample.justPressed {
		cx, cy := float64(sample.x)/scale, float64(sample.y)/scale
		t.dragging = cx >= 0 && cy >= 0 && cx <= float64(width) && cy <= float64(height)
		return "", 0, 0, false
	}
	wasDrag := t.dragging
	if sample.justReleased || !sample.pressed {
		t.dragging = false
	}
	if !moved {
		return "", 0, 0, false
	}
	cx, cy := float64(sample.x)/scale, float64(sample.y)/scale
	inside := cx >= 0 && cy >= 0 && cx <= float64(width) && cy <= float64(height)
	if wasDrag {
		return "drag", cx - float64(width)/2, float64(height)/2 - cy, true
	}
	if !inside || sample.pressed {
		return "", 0, 0, false
	}
	return "move", cx - float64(width)/2, float64(height)/2 - cy, true
}

func (w *window) Update() error {
	w.readyOnce.Do(func() {
		close(w.ready)
		// The window is up: finish AhdCode's application identity.
		ahdidentity.Apply()
	})
	if w.session.closeRequested.Load() {
		return ebiten.Termination
	}
	w.pollInput()
	return nil
}

// pollInput reports the input a program listens for: one click event per
// left-button press inside the Canvas, in Cartesian coordinates, and one key
// event per key press (auto-repeat produces no further events). Nothing is
// reported while the window is not focused.
func (w *window) pollInput() {
	s := w.session
	if !ebiten.IsFocused() || s.closed.Load() {
		return
	}
	if s.listenClick.Load() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		m := s.model
		px, py := ebiten.CursorPosition()
		x, y := float64(px)/w.scale, float64(py)/w.scale
		if x >= 0 && y >= 0 && x <= float64(m.width) && y <= float64(m.height) {
			s.emit(event{Event: "click", X: x - float64(m.width)/2, Y: float64(m.height)/2 - y})
		}
	}
	if s.listenMove.Load() || s.listenDrag.Load() {
		w.pollPointer()
	}
	if s.listenKey.Load() {
		w.keys = inpututil.AppendJustPressedKeys(w.keys[:0])
		for _, key := range w.keys {
			if name, known := keyNames[key]; known {
				s.emit(event{Event: "key", Key: name})
			}
		}
	}
}

// pollPointer reports where the pointer went since the last frame. A drag
// continues after the pointer leaves the Canvas when the press began inside.
// One event per frame is the most a program is asked to handle.
func (w *window) pollPointer() {
	s, m := w.session, w.session.model
	px, py := ebiten.CursorPosition()
	kind, x, y, ok := w.pointer.event(pointerSample{
		x: px, y: py,
		pressed:      ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		justPressed:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		justReleased: inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft),
	}, m.width, m.height, w.scale)
	if !ok {
		return
	}
	listening := s.listenMove.Load()
	if kind == "drag" {
		listening = s.listenDrag.Load()
	}
	if listening {
		s.emit(event{Event: kind, X: x, Y: y})
	}
}

func (w *window) Layout(int, int) (int, int) {
	m := w.session.model
	scale := 1.0
	if monitor := ebiten.Monitor(); monitor != nil && monitor.DeviceScaleFactor() > 0 {
		scale = monitor.DeviceScaleFactor()
	}
	if math.Ceil(float64(m.width)*scale)*math.Ceil(float64(m.height)*scale) > maxDisplayPixels {
		scale = 1
	}
	w.scale = scale
	return int(math.Ceil(float64(m.width) * scale)), int(math.Ceil(float64(m.height) * scale))
}

func (w *window) Draw(screen *ebiten.Image) {
	m := w.session.model
	changed := false
	m.mu.Lock()
	resized := w.display == nil || w.display.Bounds().Dx() != int(math.Ceil(float64(m.width)*w.scale))
	// While the program is inside a batch the window keeps its last complete
	// picture, so a clear followed by a redraw never shows as a blank frame.
	if resized || !m.held {
		if resized || w.generation != m.generation {
			w.display = newImage(m.width, m.height, w.scale, m.background)
			w.generation, w.drawn, changed = m.generation, 0, true
		}
		t := transform{width: float64(m.width), height: float64(m.height), scale: w.scale}
		for ; w.drawn < len(m.commands); w.drawn++ {
			drawCommand(w.display, t, m.commands[w.drawn])
			changed = true
		}
	}
	m.mu.Unlock()

	bounds := w.display.Bounds()
	if w.texture == nil || w.texture.Bounds().Dx() != bounds.Dx() || w.texture.Bounds().Dy() != bounds.Dy() {
		if w.texture != nil {
			w.texture.Deallocate()
		}
		w.texture = ebiten.NewImage(bounds.Dx(), bounds.Dy())
		changed = true
	}
	if changed {
		w.texture.WritePixels(w.display.Pix)
	}
	screen.DrawImage(w.texture, nil)
}

// runWindow opens the window on the main thread and serves the protocol
// beside it. It returns once the window is gone and the runtime has closed
// the pipe or sent close.
func runWindow(s *session) error {
	m := s.model
	w := &window{session: s, ready: make(chan struct{})}
	ebiten.SetWindowTitle(m.title)
	ebiten.SetWindowSize(m.width, m.height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetRunnableOnUnfocused(true)
	ahdidentity.Prepare()
	ebiten.SetTPS(windowTPS)

	served := make(chan struct{})
	started := make(chan bool, 1)
	go func() {
		defer close(served)
		select {
		case <-w.ready:
		case <-s.windowGone:
			started <- false
			return
		}
		started <- true
		if s.send(response{ID: s.openID, OK: true}) != nil {
			s.closeRequested.Store(true)
			return
		}
		s.serve()
		// The program closed the pipe or asked for close: end the window.
		s.closeRequested.Store(true)
	}()

	err := runGame(w)
	s.closed.Store(true)
	close(s.windowGone)
	s.emitClosed()
	if !<-started {
		// The window never appeared; report why instead of hanging the
		// program's Graphics.open call.
		message := "could not open a window"
		if err != nil {
			message += ": no usable display was found"
		}
		_ = s.send(response{ID: s.openID, Error: message})
		return errors.New(message)
	}
	<-served
	return nil
}

// runGame runs the window loop and turns a library panic into an error.
func runGame(w *window) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = errors.New("the window library stopped")
		}
	}()
	return ebiten.RunGame(w)
}
