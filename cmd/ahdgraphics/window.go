package main

import (
	"errors"
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
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
}

func (w *window) Update() error {
	w.readyOnce.Do(func() { close(w.ready) })
	if w.session.closeRequested.Load() {
		return ebiten.Termination
	}
	return nil
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
	rebuild := w.display == nil || w.generation != m.generation ||
		w.display.Bounds().Dx() != int(math.Ceil(float64(m.width)*w.scale))
	if rebuild {
		w.display = newImage(m.width, m.height, w.scale, m.background)
		w.generation, w.drawn, changed = m.generation, 0, true
	}
	t := transform{width: float64(m.width), height: float64(m.height), scale: w.scale}
	for ; w.drawn < len(m.commands); w.drawn++ {
		drawCommand(w.display, t, m.commands[w.drawn])
		changed = true
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
		if writeResponse(s.out, response{OK: true}) != nil {
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
	if !<-started {
		// The window never appeared; report why instead of hanging the
		// program's Graphics.open call.
		message := "could not open a window"
		if err != nil {
			message += ": no usable display was found"
		}
		_ = writeResponse(s.out, response{Error: message})
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
