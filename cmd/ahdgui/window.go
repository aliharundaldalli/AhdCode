package main

import (
	"errors"
	"math"
	"sync"

	"ahdidentity"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The window shows the image render draws and turns mouse and keyboard input
// into model changes and events. Only the window library's window, input,
// and texture upload are used; every widget is drawn by render.go.

// repeatDelay and repeatEvery (in ticks at 30 per second) repeat the editing
// keys of a TextInput while they are held. Window.onKey never repeats.
const (
	repeatDelay = 15
	repeatEvery = 2
)

type window struct {
	session *session

	readyOnce sync.Once
	ready     chan struct{}

	scale   float64
	title   string
	ticks   int
	caret   bool
	texture *ebiten.Image
	keys    []ebiten.Key
	chars   []rune
}

func (w *window) Update() error {
	w.readyOnce.Do(func() {
		close(w.ready)
		// The window is up: finish AhdCode's application identity.
		ahdidentity.Apply()
	})
	s := w.session
	if s.closeRequested.Load() {
		return ebiten.Termination
	}
	m := s.model
	m.mu.Lock()
	title := m.title
	m.mu.Unlock()
	if title != w.title {
		ebiten.SetWindowTitle(title)
		w.title = title
	}
	w.ticks++
	if w.ticks%caretBlinkTicks == 0 {
		w.caret = !w.caret
		m.mu.Lock()
		m.dirty = true
		m.mu.Unlock()
	}
	if !ebiten.IsFocused() || s.closed.Load() {
		return nil
	}
	w.pollMouse()
	w.pollKeys()
	return nil
}

func (w *window) pollMouse() {
	s := w.session
	px, py := ebiten.CursorPosition()
	x, y := int(math.Floor(float64(px)/w.scale)), int(math.Floor(float64(py)/w.scale))
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.model.press(x, y)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		s.clicked(s.model.release(x, y))
	}
}

// pollKeys edits the focused TextInput, moves the focus with Tab, activates
// the focused Button with Enter or Space and the focused Checkbox with
// Space, and reports each key press, once, to a program listening for keys.
func (w *window) pollKeys() {
	s, m := w.session, w.session.model
	w.keys = inpututil.AppendJustPressedKeys(w.keys[:0])
	for _, key := range w.keys {
		name, known := keyNames[key]
		if !known {
			continue
		}
		switch name {
		case "Tab":
			m.focusNext()
		case "Enter", "Space":
			s.clicked(m.activateFocused(name == "Space"))
		default:
			m.edit(name)
		}
		s.pressedKey(name)
	}
	for _, key := range []ebiten.Key{ebiten.KeyBackspace, ebiten.KeyDelete, ebiten.KeyArrowLeft, ebiten.KeyArrowRight} {
		held := inpututil.KeyPressDuration(key)
		if held > repeatDelay && (held-repeatDelay)%repeatEvery == 0 {
			m.edit(keyNames[key])
		}
	}
	w.chars = ebiten.AppendInputChars(w.chars[:0])
	if len(w.chars) > 0 {
		m.insert(w.chars)
	}
}

func (w *window) Layout(int, int) (int, int) {
	m := w.session.model
	scale := 1.0
	if monitor := ebiten.Monitor(); monitor != nil && monitor.DeviceScaleFactor() > 0 {
		scale = monitor.DeviceScaleFactor()
	}
	if math.Ceil(float64(m.width)*scale)*math.Ceil(float64(m.height)*scale) > maxDimension*maxDimension {
		scale = 1
	}
	if scale != w.scale {
		w.scale = scale
		m.mu.Lock()
		m.dirty = true
		m.mu.Unlock()
	}
	return int(math.Ceil(float64(m.width) * scale)), int(math.Ceil(float64(m.height) * scale))
}

func (w *window) Draw(screen *ebiten.Image) {
	m := w.session.model
	m.mu.Lock()
	if m.dirty || w.texture == nil {
		img := render(frame{model: m, scale: w.scale, caretVisible: w.caret})
		m.dirty = false
		m.mu.Unlock()
		bounds := img.Bounds()
		if w.texture == nil || w.texture.Bounds().Dx() != bounds.Dx() || w.texture.Bounds().Dy() != bounds.Dy() {
			if w.texture != nil {
				w.texture.Deallocate()
			}
			w.texture = ebiten.NewImage(bounds.Dx(), bounds.Dy())
		}
		w.texture.WritePixels(img.Pix)
	} else {
		m.mu.Unlock()
	}
	screen.DrawImage(w.texture, nil)
}

// runWindow opens the window on the main thread and serves the protocol
// beside it. It returns once the window is gone and the runtime has closed
// the pipe or sent close.
func runWindow(s *session) error {
	m := s.model
	w := &window{session: s, ready: make(chan struct{}), scale: 1, title: m.title}
	ebiten.SetWindowTitle(m.title)
	ebiten.SetWindowSize(m.width, m.height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetRunnableOnUnfocused(true)
	ahdidentity.Prepare()
	ebiten.SetTPS(30)

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
	close(s.windowGone)
	s.emitClosed()
	if !<-started {
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
