package main

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// session serves one Canvas after its open request. In window mode the
// protocol runs on its own goroutine while the window owns the main thread;
// headless mode has no window and serves on the main goroutine.
type session struct {
	model    *model
	in       *bufio.Reader
	out      *bufio.Writer
	headless bool

	// closed is true once the Canvas can no longer be drawn on: the user
	// closed its window, wait returned, or close was requested.
	closed atomic.Bool
	// closeRequested asks the window to end itself.
	closeRequested atomic.Bool
	// windowGone is closed when the window no longer exists.
	windowGone chan struct{}

	// writeMu serializes the one output stream shared by responses, written
	// by the protocol goroutine, and events, written by the window.
	writeMu sync.Mutex
	// listenClick and listenKey are set by listen requests; until then the
	// window reports no input at all.
	listenClick atomic.Bool
	listenKey   atomic.Bool
	// closedSent makes the closed event a one-time notice.
	closedSent atomic.Bool
	// script is the headless test Canvas's event list, replayed on wait.
	script []event
	// openID is the id of the open request, answered once the window is up.
	openID int64
}

const closedMessage = "the Canvas is closed"

// send writes one protocol line: a response or an event.
func (s *session) send(value any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return writeLine(s.out, value)
}

// emit writes an event; a failed write means the program is gone, and the
// window ends.
func (s *session) emit(value event) {
	if s.send(value) != nil {
		s.closeRequested.Store(true)
	}
}

// emitClosed tells the program, once, that the Canvas window is gone.
func (s *session) emitClosed() {
	if s.closedSent.CompareAndSwap(false, true) {
		s.emit(event{Event: "closed"})
	}
}

// serve answers requests until the runtime closes the pipe or sends close.
func (s *session) serve() {
	for {
		value, err := readRequest(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			_ = s.send(response{Error: err.Error()})
			if errors.Is(err, errRequestTooLarge) {
				// The rest of that line cannot be told apart from the next
				// request, so the stream is not trusted any further.
				return
			}
			continue
		}
		reply, stop := s.handle(value)
		reply.ID = value.ID
		if s.send(reply) != nil || stop {
			return
		}
		if value.Op == "wait" && s.headless {
			// A headless Canvas has no window to close: it replays its test
			// script, if any, and reports itself closed at once.
			for _, scripted := range s.script {
				s.emit(scripted)
			}
			s.emitClosed()
		}
	}
}

func (s *session) handle(value request) (response, bool) {
	switch value.Op {
	case "status":
		open := !s.closed.Load()
		return response{OK: true, Open: &open}, false
	case "close":
		s.requestClose()
		return response{OK: true}, true
	case "wait":
		// wait no longer blocks the protocol: the program keeps sending
		// requests from its event callbacks and learns that the window is
		// gone from the closed event.
		return response{OK: true}, false
	case "listen":
		switch value.Kind {
		case "click":
			s.listenClick.Store(true)
		case "key":
			s.listenKey.Store(true)
		default:
			return response{Error: "unknown event kind " + value.Kind}, false
		}
		return response{OK: true}, false
	case "save":
		if s.closed.Load() {
			return response{Error: closedMessage, Closed: true}, false
		}
		if err := s.save(value.Path); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	case "clear", "line", "circle", "rectangle":
		if s.closed.Load() {
			return response{Error: closedMessage, Closed: true}, false
		}
		if err := s.model.apply(value); err != nil {
			return response{Error: err.Error()}, false
		}
		return response{OK: true}, false
	}
	return response{Error: "unknown request " + value.Op}, false
}

// requestClose ends the window, if any, and waits briefly for it to go.
func (s *session) requestClose() {
	s.closed.Store(true)
	if s.headless {
		return
	}
	s.closeRequested.Store(true)
	select {
	case <-s.windowGone:
	case <-time.After(5 * time.Second):
	}
}

func (s *session) save(path string) error {
	format, err := exportFormat(path)
	if err != nil {
		return err
	}
	snap := s.model.snapshot()
	var data []byte
	if format == "png" {
		if data, err = encodePNG(snap); err != nil {
			return errors.New("could not encode the PNG image")
		}
	} else {
		data = encodeSVG(snap)
	}
	if err := os.WriteFile(path, data, 0o666); err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			return errors.New("could not write the file: " + pathError.Err.Error())
		}
		return errors.New("could not write the file")
	}
	return nil
}
